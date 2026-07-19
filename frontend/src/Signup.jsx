/* eslint-disable no-unused-vars */
import React, { useState } from 'react';
import { ToastContainer, toast } from 'react-toastify';
import 'react-toastify/dist/ReactToastify.css';
import { FaEnvelope, FaLock, FaEye, FaEyeSlash } from 'react-icons/fa';
import { motion } from 'framer-motion';
import { useAuth } from './AuthContext';
import Lottie from 'lottie-react';
import foxAnimation from './assets/fox-animation.json';
import { Particles } from 'react-tsparticles';
import { loadFull } from 'tsparticles';

const Signup = () => {
    const [state, setState] = useState("Sign Up");
    const [email, setEmail] = useState("");
    const [password, setPassword] = useState("");
    const [showPassword, setShowPassword] = useState(false);
    const [loading, setLoading] = useState(false);
    const { api, login } = useAuth();

    const particlesInit = async (engine) => {
        await loadFull(engine);
    };

            const onSubmitHandler = async (e) => {
            e.preventDefault();
            setLoading(true);
            try {
                if (state === 'Sign Up') {
                    await api.post('/api/v1/auth/signup', { email, password });
                    toast.success("Account created successfully!");
                    setState('Login');
                    setEmail("");
                    setPassword("");
                } else {
                     await login({ email, password });
                }
            } catch (error) {
                toast.error(error.response?.data?.error || error.message);
            } finally {
                setLoading(false);
            }
        };

    return (
        <div className="min-h-screen bg-gray-900 overflow-hidden relative">
            <div className="absolute inset-0 z-0">
                <Particles
                    init={particlesInit}
                    options={{
                        fullScreen: false,
                        background: {
                            color: "#0a0e17",
                        },
                        particles: {
                            number: {
                                value: 120,
                                density: {
                                    enable: true,
                                    value_area: 800
                                }
                            },
                            color: {
                                value: ["#ffffff", "#d1d5db", "#9ca3af"]
                            },
                            shape: {
                                type: "circle",
                            },
                            opacity: {
                                value: 0.7,
                                random: true,
                            },
                            size: {
                                value: { min: 1, max: 3 },
                                random: true,
                            },
                            move: {
                                enable: true,
                                speed: 0.3,
                            }
                        }
                    }}
                />
            </div>

            <div className="absolute left-10 bottom-10 w-64 h-64 z-10 opacity-90">
                <Lottie animationData={foxAnimation} loop={true} />
            </div>

            <div className="relative z-10 min-h-screen flex items-center justify-center p-4">
                <motion.div 
                    initial={{ opacity: 0, y: 20 }}
                    animate={{ opacity: 1, y: 0 }}
                    transition={{ duration: 0.5 }}
                    className="relative w-full max-w-md"
                >
                    <div className="bg-gray-800/60 backdrop-blur-lg rounded-xl shadow-xl border border-gray-700/30 p-8">
                        <div className="text-center mb-8">
                            <h2 className="text-3xl font-bold text-transparent bg-clip-text bg-gradient-to-r from-blue-400 to-purple-400 mb-2">
                                {state === 'Sign Up' ? "Join CodeHunt" : "Welcome Back"}
                            </h2>
                            <p className="text-gray-400">
                                {state === 'Sign Up' ? "Create your account to continue" : "Login to access your dashboard"}
                            </p>
                        </div>

                        <form onSubmit={onSubmitHandler} className="space-y-5">
                            <div className="relative">
                                <label className="block text-sm font-medium text-gray-300 mb-1">Email</label>
                                <div className="relative">
                                    <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
                                        <FaEnvelope className="text-gray-400" />
                                    </div>
                                    <input
                                        className="w-full pl-10 pr-4 py-2.5 bg-gray-700/40 rounded-lg focus:ring-2 focus:ring-blue-500/50 text-gray-100 placeholder-gray-400"
                                        type="email"
                                        placeholder="your@email.com"
                                        onChange={(e) => setEmail(e.target.value)}
                                        value={email}
                                        required
                                    />
                                </div>
                            </div>

                            <div className="relative">
                                <label className="block text-sm font-medium text-gray-300 mb-1">Password</label>
                                <div className="relative">
                                    <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
                                        <FaLock className="text-gray-400" />
                                    </div>
                                    <input
                                        className="w-full pl-10 pr-12 py-2.5 bg-gray-700/40 rounded-lg focus:ring-2 focus:ring-blue-500/50 text-gray-100 placeholder-gray-400"
                                        type={showPassword ? "text" : "password"}
                                        placeholder="••••••••"
                                        onChange={(e) => setPassword(e.target.value)}
                                        value={password}
                                        required
                                    />
                                    <button
                                        type="button"
                                        className="absolute inset-y-0 right-0 pr-3 flex items-center"
                                        onClick={() => setShowPassword(!showPassword)}
                                    >
                                        {showPassword ? (
                                            <FaEyeSlash className="text-gray-400 hover:text-gray-300" />
                                        ) : (
                                            <FaEye className="text-gray-400 hover:text-gray-300" />
                                        )}
                                    </button>
                                </div>
                            </div>

                            <button
                                type="submit"
                                className="w-full bg-gradient-to-r from-blue-600 to-purple-600 text-white py-3 px-4 rounded-lg font-medium flex items-center justify-center mt-6"
                                disabled={loading}
                            >
                                {loading ? (
                                    <span className="flex items-center">
                                        <svg className="animate-spin -ml-1 mr-3 h-5 w-5 text-white" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                                            <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
                                            <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                                        </svg>
                                        Processing...
                                    </span>
                                ) : (
                                    <span>{state === 'Sign Up' ? "Create Account" : "Sign In"}</span>
                                )}
                            </button>
                        </form>

                        <div className="text-center mt-6 text-sm text-gray-400">
                            {state === 'Sign Up' ? (
                                <p>Already have an account?{' '}
                                    <button 
                                        onClick={() => setState('Login')} 
                                        className="text-blue-400 hover:text-blue-300"
                                    >
                                        Sign in
                                    </button>
                                </p>
                            ) : (
                                <p>New to CodeHunt?{' '}
                                    <button 
                                        onClick={() => setState('Sign Up')} 
                                        className="text-purple-400 hover:text-purple-300"
                                    >
                                        Create account
                                    </button>
                                </p>
                            )}
                        </div>
                    </div>
                </motion.div>
            </div>

            <ToastContainer position="top-right" theme="dark" />
        </div>
    );
};

export default Signup;
