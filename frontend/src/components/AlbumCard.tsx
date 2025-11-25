import React from 'react';
import { motion } from 'framer-motion';
import { Check, Circle, Lock } from 'lucide-react';
import { Album } from '../types';
import clsx from 'clsx';

interface AlbumCardProps {
    album: Album;
    isSelected: boolean;
    onToggle: () => void;
}

export const AlbumCard: React.FC<AlbumCardProps> = ({ album, isSelected, onToggle }) => {
    const isPaid = album.status === 'paid';

    return (
        <motion.div
            layout
            initial={{ opacity: 0, scale: 0.9 }}
            animate={{ opacity: isPaid ? 0.5 : 1, scale: 1 }}
            whileHover={{ scale: isPaid ? 1 : 1.02 }}
            className={clsx(
                "relative flex items-center p-3 rounded-xl border transition-all overflow-hidden group",
                isPaid ? "cursor-default border-slate-800 bg-surface/50" : "cursor-pointer bg-surface border-slate-700 hover:border-slate-600",
                isSelected && !isPaid
                    ? "bg-primary/10 border-primary shadow-[0_0_15px_rgba(85,96,255,0.3)]"
                    : ""
            )}
            onClick={!isPaid ? onToggle : undefined}
        >
            {/* Cover Image */}
            <div className="relative w-16 h-16 rounded-lg overflow-hidden flex-shrink-0 mr-4 shadow-md">
                <img src={album.coverUrl} alt={album.title} className="w-full h-full object-cover" />
            </div>

            {/* Info */}
            <div className="flex-1 min-w-0">
                <div className="relative group/tooltip">
                    <h3 className="text-white font-medium text-sm truncate pr-2 cursor-default" title={album.title}>
                        {album.title}
                    </h3>
                    {/* Custom Tooltip */}
                    <div className="absolute bottom-full left-0 mb-2 hidden group-hover/tooltip:block z-50 w-max max-w-[250px] bg-slate-900 text-white text-xs p-2 rounded shadow-xl border border-slate-700 pointer-events-none whitespace-normal break-words">
                        {album.title}
                    </div>
                </div>
                <p className="text-slate-400 text-xs truncate">{album.url}</p>

                {/* Status Badge */}
                <div className="mt-2 flex items-center space-x-2">
                    <span className={clsx(
                        "text-[10px] px-2 py-0.5 rounded-full font-medium uppercase tracking-wider",
                        isPaid
                            ? "bg-slate-700 text-slate-400"
                            : "bg-emerald-500/20 text-emerald-400"
                    )}>
                        {isPaid ? "Paid" : "Free / NYP"}
                    </span>
                </div>
            </div>

            {/* Selection Indicator */}
            <div className="flex-shrink-0 ml-2">
                {isPaid ? (
                    <Lock className="w-5 h-5 text-slate-600" />
                ) : isSelected ? (
                    <div className="w-6 h-6 rounded-full bg-primary flex items-center justify-center shadow-lg shadow-primary/50">
                        <Check className="w-4 h-4 text-white" />
                    </div>
                ) : (
                    <Circle className="w-6 h-6 text-slate-600 group-hover:text-slate-500 transition-colors" />
                )}
            </div>
        </motion.div>
    );
};
