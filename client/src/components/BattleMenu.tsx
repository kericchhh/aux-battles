import React, { useState } from 'react';

const BattleMenu = () => {
    const [activeTab, setActiveTab] = useState('create'); // 'create' | 'join'
    const [rounds, setRounds] = useState('');
    const [inviteCode, setInviteCode] = useState('');

    const handleCreate = () => {
        console.log('Creating battle with rounds:', rounds);
        // Add creation logic here
    };

    const handleJoin = () => {
        console.log('Joining with code:', inviteCode);
        // Add join logic here
    };

    return (
        <div className="flex items-center justify-center w-full h-full">
            <div className="w-[450px] rounded-2xl overflow-hidden border border-[#5964a6] bg-[#090a11] text-[#f4f0f7] font-sans shadow-lg backdrop-blur-sm">
                <div className="flex">
                    <button
                        onClick={() => setActiveTab('create')}
                        className={`flex-1 py-4 text-center transition-colors ${activeTab === 'create'
                                ? 'bg-[#5964a6] font-semibold text-[#f4f0f7]'
                                : 'hover:bg-[#959cc6] text-[#f4f0f7]'
                            }`}
                    >
                        Create a room
                    </button>

                    <button
                        onClick={() => setActiveTab('join')}
                        className={`flex-1 py-4 text-center transition-colors ${activeTab === 'join'
                                ? 'bg-[#5964a6] font-semibold text-[#f4f0f7]'
                                : 'hover:bg-[#959cc6] text-[#f4f0f7]'
                            }`}
                    >
                        Join with a code
                    </button>
                </div>

                <div className="p-10 flex flex-col items-center justify-center min-h-[300px]">

                    {activeTab === 'create' && (
                        <div className="flex flex-col items-center w-full space-y-12 animate-fade-in">
                            <input
                                type="number"
                                max="10"
                                min="1"
                                value={rounds}
                                onChange={(e) => setRounds(e.target.value)}
                                placeholder="Number of Rounds: max(10)"
                                className="w-full bg-transparent border border-gray-300 rounded-xl px-4 py-4 text-center text-white placeholder-gray-400 focus:outline-none focus:border-white focus:ring-1 focus:ring-white transition-all"
                            />
                            <button
                                onClick={handleCreate}
                                className="rounded-xl px-8 py-3 text-sm font-medium tracking-wider uppercase bg-[#5964a6] hover:bg-[#959cc6] hover:text-black transition-colors"
                            >
                                Create Battle
                            </button>
                        </div>
                    )}

                    {activeTab === 'join' && (
                        <div className="flex flex-col items-center w-full space-y-12 animate-fade-in">
                            <input
                                type="text"
                                value={inviteCode}
                                onChange={(e) => setInviteCode(e.target.value)}
                                placeholder="Enter invite code:"
                                className="w-full bg-transparent rounded-xl px-4 py-4 text-center text-white placeholder-gray-400 focus:outline-none focus:border-white focus:ring-1 focus:ring-white transition-all"
                            />
                            <button
                                onClick={handleJoin}
                                className="rounded-xl px-12 py-3 text-sm font-medium tracking-wider ] bg-[#5964a6] hover:bg-[#959cc6] hover:text-black transition-colors"
                            >
                                Join
                            </button>
                        </div>
                    )}

                </div>
            </div>
        </div>
    );
};

export default BattleMenu;
