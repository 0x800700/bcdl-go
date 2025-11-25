import React from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { CheckCircle } from 'lucide-react';

interface StatusPanelProps {
    show: boolean;
    downloadedCount: number;
    failedCount: number;
    skippedCount: number;
}

export const StatusPanel: React.FC<StatusPanelProps> = ({ show, downloadedCount, failedCount, skippedCount }) => {
    return (
        <AnimatePresence>
            {show && (
                <motion.div
                    initial={{ opacity: 0, y: -20 }}
                    animate={{ opacity: 1, y: 0 }}
                    exit={{ opacity: 0, y: -20 }}
                    className="bg-surface border border-emerald-500/30 rounded-xl p-4 mb-6 flex items-center shadow-lg shadow-emerald-500/10"
                >
                    <div className="w-10 h-10 rounded-full bg-emerald-500/20 flex items-center justify-center mr-4">
                        <CheckCircle className="w-6 h-6 text-emerald-500" />
                    </div>
                    <div>
                        <h3 className="text-emerald-400 font-bold text-sm">Download Complete</h3>
                        <p className="text-slate-400 text-xs mt-0.5">
                            Downloaded: <span className="text-white">{downloadedCount}</span> •
                            Failed: <span className="text-red-400">{failedCount}</span> •
                            Skipped: <span className="text-amber-400">{skippedCount}</span>
                        </p>
                    </div>
                </motion.div>
            )}
        </AnimatePresence>
    );
};
