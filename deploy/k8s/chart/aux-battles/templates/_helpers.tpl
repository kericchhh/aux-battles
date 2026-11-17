{{- define "aux-battles.labels" -}}
app.kubernetes.io/part-of: aux-battles
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/version: {{ .Values.image.tag | default .Chart.AppVersion | quote }}
{{- end }}
