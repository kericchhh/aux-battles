package server

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/kericchhh/aux-battles/internal/dbgen"
	"github.com/kericchhh/aux-battles/internal/media"
)

func (s *server) persistSongUpload(ctx context.Context, in validatedSongUpload) (dbgen.CreateSongRow, error) {
	return s.persistSongUploadWithBegin(ctx, in, s.pool.Begin)
}

func (s *server) persistSongUploadWithBegin(ctx context.Context, in validatedSongUpload, begin func(context.Context) (pgx.Tx, error)) (dbgen.CreateSongRow, error) {
	var empty dbgen.CreateSongRow
	keepFile := false
	defer func() {
		if !keepFile {
			if err := media.FromEnv().RemoveInput(in.Path); err != nil {
				s.loggerFor(ctx).Error("remove failed song upload", "path", in.Path, "error", err)
			}
		}
	}()

	tx, err := begin(ctx)
	if err != nil {
		s.loggerFor(ctx).Error("begin song upload transaction", "error", err)
		return empty, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	q := s.queries.WithTx(tx)
	song, err := q.CreateSong(ctx, dbgen.CreateSongParams{
		Title: in.Title, Artist: in.Artist, Genre: in.Genre,
		DurationSeconds: in.DurationSeconds,
	})
	if err != nil {
		s.loggerFor(ctx).Error("create song", "error", err)
		return empty, err
	}

	var songID pgtype.UUID
	if err := songID.Scan(song.ID); err != nil {
		s.loggerFor(ctx).Error("parse created song ID", "error", err)
		return empty, err
	}
	if err := q.EnqueueSongJob(ctx, dbgen.EnqueueSongJobParams{
		SongID: songID, InputPath: in.Path, ClipStartSeconds: in.ClipStartSeconds,
	}); err != nil {
		s.loggerFor(ctx).Error("enqueue song", "error", err)
		return empty, err
	}
	if err := tx.Commit(ctx); err != nil {
		if !errors.Is(err, pgx.ErrTxCommitRollback) {
			keepFile = true
			s.loggerFor(ctx).Info("song upload commit outcome unknown; retaining file", "song_id", song.ID, "path", in.Path)
		}
		s.loggerFor(ctx).Error("commit song upload", "error", err)
		return empty, err
	}
	keepFile = true
	return song, nil
}
