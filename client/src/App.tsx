import { BrowserRouter, Routes, Route } from "react-router-dom";
import { AuthProvider } from "./context/AuthContext";
import NavBar from "./components/Navbar"

import Login from "./pages/Login";
import Register from "./pages/Register";
import Lobby from "./pages/Lobby";
import Battle from "./pages/Battle";
import Profile from "./pages/Profile";
import ProtectedRoute from "./components/ProtectedRoute";

export default function App() {
    return (
        <AuthProvider>
            <BrowserRouter>
                <div className="h-screen w-screen overflow-hidden bg-[#0d1020]">
                    <Routes>
                        <Route path="/login" element={<Login />} />
                        <Route path="/register" element={<Register />} />

                        <Route
                            path="/"
                            element={
                                <ProtectedRoute>
                                    <div className="flex h-screen flex-col overflow-hidden">
                                        <NavBar />

                                        <main className="min-h-0 flex-1 overflow-hidden">
                                            <Lobby />
                                        </main>
                                    </div>
                                </ProtectedRoute>
                            }
                        />

                        <Route
                            path="/battle/:battleId"
                            element={
                                <ProtectedRoute>
                                    <div className="flex h-screen flex-col overflow-hidden">
                                        <NavBar />

                                        <main className="min-h-0 flex-1 overflow-hidden">
                                            <Battle />
                                        </main>
                                    </div>
                                </ProtectedRoute>
                            }
                        />

                        <Route
                            path="/profile/:id"
                            element={
                                <ProtectedRoute>
                                    <div className="flex h-screen flex-col">
                                        <NavBar />
                                        <main className="min-h-0 flex-1">
                                            <Profile />
                                        </main>
                                    </div>
                                </ProtectedRoute>
                            }
                        />
                    </Routes>
                </div>
            </BrowserRouter>
        </AuthProvider>
    );
}
