import React from 'react';
import { Search, FolderOpen, Download, X } from 'lucide-react';
import logoImage from '../assets/ProBablyWorks.png';

interface SidebarProps {
    url: string;
    setUrl: (url: string) => void;
    folder: string;
    onSelectFolder: () => void;
    onScan: () => void;
    onStop: () => void;
    isScanning: boolean;
}

export const Sidebar: React.FC<SidebarProps> = ({
    url, setUrl, folder, onSelectFolder, onScan, onStop, isScanning
}) => {
    return (
        <div className="w-80 bg-surface border-r border-slate-700 p-6 flex flex-col h-full">
            <h1 className="text-2xl font-bold text-white mb-8 flex items-center">
                <Download className="w-6 h-6 mr-3 text-primary" />
                Bandcamp DL
            </h1>

            {/* Artist URL Input */}
            <div className="mb-6">
                <label className="block text-slate-400 text-xs uppercase font-bold mb-2 tracking-wider">
                    Artist URL
                </label>
                <div className="relative">
                    <input
                        type="text"
                        value={url}
                        onChange={(e) => setUrl(e.target.value)}
                        placeholder="https://artist.bandcamp.com"
                        className="w-full bg-background border border-slate-700 rounded-lg py-3 px-4 pl-10 text-sm text-white focus:outline-none focus:border-primary focus:ring-1 focus:ring-primary transition-all"
                    />
                    <Search className="absolute left-3 top-3.5 w-4 h-4 text-slate-500" />
                </div>
            </div>

            {/* Scan/Stop Button */}
            {isScanning ? (
                <button
                    onClick={onStop}
                    className="w-full bg-red-500 hover:bg-red-600 text-white font-medium py-3 rounded-lg transition-all shadow-lg shadow-red-500/25 flex items-center justify-center mb-8"
                >
                    <X className="w-4 h-4 mr-2" />
                    Stop Scan
                </button>
            ) : (
                <button
                    onClick={onScan}
                    disabled={!url}
                    className="w-full bg-primary hover:bg-blue-600 disabled:opacity-50 disabled:cursor-not-allowed text-white font-medium py-3 rounded-lg transition-all shadow-lg shadow-primary/25 flex items-center justify-center mb-8"
                >
                    <Search className="w-4 h-4 mr-2" />
                    Scan Artist
                </button>
            )}

            {/* Download Folder */}
            <div className="mb-6">
                <label className="block text-slate-400 text-xs uppercase font-bold mb-2 tracking-wider">
                    Download Folder
                </label>
                <div className="flex gap-2">
                    <div className="flex-1 bg-background border border-slate-700 rounded-lg py-3 px-4 text-sm text-slate-300 truncate">
                        {folder || "Select folder..."}
                    </div>
                    <button
                        onClick={onSelectFolder}
                        className="bg-slate-700 hover:bg-slate-600 text-white p-3 rounded-lg transition-colors"
                    >
                        <FolderOpen className="w-5 h-5" />
                    </button>
                </div>
            </div>

            <div className="mt-auto">
                <img src={logoImage} alt="ProBably Works" className="w-32 opacity-80 hover:opacity-100 transition-opacity" />
            </div>
        </div>
    );
};
