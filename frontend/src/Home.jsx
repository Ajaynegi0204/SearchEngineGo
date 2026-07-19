/* eslint-disable no-unused-vars */
import React, { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { motion, AnimatePresence } from 'framer-motion';
import Lottie from 'lottie-react';
import foxAnimation from './assets/fox-animation.json';
import { FaSearch, FaCode, FaChartLine, FaArrowRight } from 'react-icons/fa';
import { useAuth } from './AuthContext';

const Home = () => {
    const [isLoading, setIsLoading] = useState(true);
    const { isLoggedIn } = useAuth();
    const navigate = useNavigate();

    useEffect(() => {
        const timer = setTimeout(() => {
            setIsLoading(false);
            if (isLoggedIn) {
                navigate('/dashboard');
            }
        }, 1500);
        return () => clearTimeout(timer);
    }, [isLoggedIn, navigate]);

    const handleNavigation = () => {
        navigate(isLoggedIn ? '/dashboard' : '/signup');
    };

    return (
        <>
            <AnimatePresence>
                {isLoading && (
                    <motion.div
                        initial={{ opacity: 1 }}
                        exit={{ opacity: 0 }}
                        transition={{ duration: 0.5 }}
                        className="fixed inset-0 bg-gray-900 z-50 flex flex-col items-center justify-center"
                    >
                        <motion.div
                            animate={{ 
                                rotate: 360,
                                scale: [1, 1.1, 1]
                            }}
                            transition={{ 
                                rotate: { 
                                    duration: 1.5, 
                                    repeat: Infinity, 
                                    ease: "linear" 
                                },
                                scale: {
                                    duration: 1.2,
                                    repeat: Infinity,
                                    repeatType: "reverse"
                                }
                            }}
                            className="w-16 h-16 mb-4"
                        >
                            <div className="w-full h-full rounded-full border-4 border-t-purple-500 border-r-blue-500 border-b-purple-500 border-l-blue-500"></div>
                        </motion.div>
                        <motion.h1
                            initial={{ opacity: 0, y: 10 }}
                            animate={{ opacity: 1, y: 0 }}
                            transition={{ delay: 0.2 }}
                            className="text-2xl font-bold bg-clip-text text-transparent bg-gradient-to-r from-blue-400 to-purple-400"
                        >
                            CodeHunt
                        </motion.h1>
                    </motion.div>
                )}
            </AnimatePresence>

            <div className="min-h-screen bg-gradient-to-br from-gray-900 to-gray-800 overflow-hidden relative">
                {/* Animated Fox */}
                <motion.div 
                    initial={{ opacity: 0, x: 100 }}
                    animate={{ opacity: 1, x: 0 }}
                    transition={{ delay: 0.5, duration: 0.8 }}
                    className="absolute right-10 bottom-10 w-64 h-64 z-10 opacity-90 hover:opacity-100 transition-opacity"
                >
                    <Lottie animationData={foxAnimation} loop={true} />
                </motion.div>

                <div className="relative z-10 min-h-screen flex flex-col">
                    <nav className="flex justify-between items-center p-6">
                        <motion.div 
                            initial={{ opacity: 0, x: -20 }}
                            animate={{ opacity: 1, x: 0 }}
                            transition={{ delay: 0.4 }}
                            className="flex items-center group cursor-pointer"
                            onClick={() => navigate('/')}
                        >
                            <div className="w-8 h-8 rounded-lg bg-gradient-to-r from-blue-500 to-purple-500 mr-2 flex items-center justify-center">
                                <FaCode className="text-white text-sm" />
                            </div>
                            <span className="text-xl font-bold bg-clip-text text-transparent bg-gradient-to-r from-blue-400 to-purple-400 group-hover:from-blue-300 group-hover:to-purple-300 transition-all">
                                CodeHunt
                            </span>
                        </motion.div>
                        <motion.div
                            initial={{ opacity: 0, x: 20 }}
                            animate={{ opacity: 1, x: 0 }}
                            transition={{ delay: 0.4 }}
                        >
                            <button
                                onClick={handleNavigation}
                                className="px-4 py-2 rounded-lg bg-gradient-to-r from-blue-600 to-purple-600 text-white hover:shadow-lg hover:shadow-blue-500/30 transition-all flex items-center"
                            >
                                {isLoggedIn ? 'Go to Dashboard' : 'Get Started'}
                                <FaArrowRight className="ml-2" />
                            </button>
                        </motion.div>
                    </nav>

                    <div className="flex-1 flex flex-col items-center justify-center px-6 text-center">
                        <motion.div 
                            initial={{ opacity: 0, y: -20 }}
                            animate={{ opacity: 1, y: 0 }}
                            transition={{ delay: 0.6 }}
                            className="max-w-3xl"
                        >
                            <h1 className="text-5xl md:text-6xl font-bold mb-6 bg-clip-text text-transparent bg-gradient-to-r from-blue-300 to-purple-400 leading-tight">
                                The Smart Way to <br />Find Coding Problems
                            </h1>
                            <p className="text-xl text-gray-300 mb-10">
                                Search thousands of LeetCode problems using keyword matching, embeddings, and reranking.
                            </p>
                            <div className="flex justify-center">
                                <button
                                    onClick={handleNavigation}
                                    className="px-8 py-3.5 rounded-lg bg-gradient-to-r from-blue-600 to-purple-600 text-white font-medium flex items-center hover:shadow-lg hover:shadow-blue-500/30 transition-all hover:scale-[1.02]"
                                >
                                    {isLoggedIn ? 'Continue to Dashboard' : 'Start Hunting'} 
                                    <FaArrowRight className="ml-2" />
                                </button>
                            </div>
                        </motion.div>

                        {!isLoggedIn && (
                            <motion.div 
                                initial={{ opacity: 0 }}
                                animate={{ opacity: 1 }}
                                transition={{ delay: 0.8 }}
                                className="grid grid-cols-1 md:grid-cols-3 gap-8 max-w-5xl w-full mt-20 px-4"
                            >
                                {[
                                    {
                                        icon: <FaSearch className="text-3xl mb-4 text-blue-400" />,
                                        title: "Semantic Search",
                                        desc: "Understands coding terms like 'DSU' or 'sliding window' automatically"
                                    },
                                    {
                                        icon: <FaCode className="text-3xl mb-4 text-purple-400" />,
                                        title: "Custom Engine",
                                        desc: "BM25 finds important algorithm and topic terms"
                                    },
                                    {
                                        icon: <FaChartLine className="text-3xl mb-4 text-blue-400" />,
                                        title: "Vector Powered",
                                        desc: "Embeddings and reranking find problems by intent"
                                    }
                                ].map((feature, i) => (
                                    <motion.div 
                                        key={i}
                                        whileHover={{ y: -5 }}
                                        whileTap={{ scale: 0.98 }}
                                        className="bg-gray-800/50 backdrop-blur-lg p-6 rounded-xl border border-gray-700/30 hover:border-gray-600/50 transition-all cursor-default"
                                    >
                                        <div className="flex flex-col items-center text-center">
                                            {feature.icon}
                                            <h3 className="text-xl font-bold mb-2 text-gray-100">{feature.title}</h3>
                                            <p className="text-gray-400">{feature.desc}</p>
                                        </div>
                                    </motion.div>
                                ))}
                            </motion.div>
                        )}
                    </div>
                </div>
            </div>
        </>
    );
};

export default Home;
