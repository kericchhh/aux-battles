import BattleMenu from "../components/BattleMenu";
export default function Lobby() {
    return (
        <div
            className="
        min-h-screen
        w-full
        flex
        flex-col
        items-center
        justify-center
        bg-app
      "
        >
            <BattleMenu />
        </div>
    );
}
