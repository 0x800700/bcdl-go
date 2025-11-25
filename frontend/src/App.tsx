import { useState, useEffect } from 'react';
import { Sidebar } from './components/Sidebar';
import { AlbumCard } from './components/AlbumCard';
import { LogPanel } from './components/LogPanel';
import { StatusPanel } from './components/StatusPanel';
import { Album, LogMessage } from './types';
import { EventsOn } from '../wailsjs/runtime/runtime';
import { ScanArtist, SelectFolder, DownloadAlbum, StopScan } from '../wailsjs/go/main/App';

function App() {
    const [url, setUrl] = useState("");
    const [folder, setFolder] = useState("");
    const [albums, setAlbums] = useState<Album[]>([]);
    const [selectedAlbums, setSelectedAlbums] = useState<Set<string>>(new Set());
    const [logs, setLogs] = useState<LogMessage[]>([]);
    const [isScanning, setIsScanning] = useState(false);
    const [isDownloading, setIsDownloading] = useState(false);
    const [showStatus, setShowStatus] = useState(false);

    // Stats
    const [downloadedCount, setDownloadedCount] = useState(0);
    const [failedCount, setFailedCount] = useState(0);
    const [skippedCount, setSkippedCount] = useState(0);

    const addLog = (message: string, type: LogMessage['type'] = 'info') => {
        setLogs(prev => [...prev, {
            timestamp: new Date().toLocaleTimeString(),
            message,
            type
        }]);
    };

    useEffect(() => {
        // Wails Event Listeners
        try {
            EventsOn("scan:start", (url: string) => {
                setIsScanning(true);
                addLog(`Scanning artist: ${url}`, 'info');
            });

            EventsOn("scan:complete", (results: Album[]) => {
                setIsScanning(false);
                setAlbums(results || []);
                addLog(`Found ${results?.length || 0} albums`, 'success');
            });

            const cleanupFound = EventsOn("scan:album_found", (album: Album) => {
                console.log("Received scan:album_found:", album.title);
                addLog(`Found album: ${album.title}`, 'info');
                setAlbums(prev => {
                    // Avoid duplicates just in case
                    if (prev.some(a => a.url === album.url)) return prev;
                    return [...prev, album];
                });
            });

            EventsOn("scan:error", (err: string) => {
                setIsScanning(false);
                addLog(`Scan error: ${err}`, 'error');
            });

            EventsOn("scan:stopped", (count: number) => {
                setIsScanning(false);
                addLog(`Scan stopped. Found ${count} albums.`, 'warning');
            });

            EventsOn("download:start", (url: string) => {
                addLog(`Starting download: ${url}`, 'info');
            });

            EventsOn("download:progress", (data: any) => {
                // Optional: Update specific album progress if needed
                // For now just log it
                // addLog(data.message, 'info'); 
            });

            EventsOn("download:complete", (url: string) => {
                setDownloadedCount(prev => prev + 1);
                addLog(`Download complete: ${url}`, 'success');
            });

            EventsOn("download:error", (data: any) => {
                setFailedCount(prev => prev + 1);
                addLog(`Download failed: ${data.error}`, 'error');
            });

            EventsOn("log:error", (msg: string) => addLog(msg, 'error'));
        } catch (e) {
            console.warn("Wails runtime not available. Events disabled.");
            addLog("Running in browser mode (No Wails backend)", 'warning');
        }

        return () => {
            // Wails runtime handles cleanup, but good practice if we could
            // For Wails, these are generally not needed as events persist
            // cleanupStart();
            // cleanupFound();
            // cleanupComplete();
            // cleanupError();
        };
    }, []);

    const handleScan = async () => {
        if (!url) return;
        console.log('Starting scan for:', url);
        setAlbums([]);
        setSelectedAlbums(new Set());
        setShowStatus(false);
        setIsScanning(true);
        // Don't addLog here - the scan:start event will do it

        try {
            // We ignore the return value here as we rely on events for dynamic updates
            await ScanArtist(url);
        } catch (err) {
            console.error('Scan error:', err);
            setIsScanning(false);
            // Don't addLog here - the scan:error event will do it
        }
    };

    const handleStopScan = async () => {
        try {
            await StopScan();
        } catch (err) {
            console.error('Stop scan error:', err);
        }
    };

    const handleSelectFolder = async () => {
        try {
            const path = await SelectFolder();
            if (path) {
                setFolder(path);
                addLog(`Selected folder: ${path}`, 'info');
            }
        } catch (err) {
            addLog(`Error selecting folder: ${err}`, 'error');
        }
    };

    const handleToggleAlbum = (albumUrl: string) => {
        const newSelected = new Set(selectedAlbums);
        if (newSelected.has(albumUrl)) {
            newSelected.delete(albumUrl);
        } else {
            newSelected.add(albumUrl);
        }
        setSelectedAlbums(newSelected);
    };

    const handleDownload = async () => {
        if (!folder) {
            addLog("Please select a download folder first", 'warning');
            return;
        }
        if (selectedAlbums.size === 0) {
            addLog("No albums selected", 'warning');
            return;
        }

        setIsDownloading(true);
        setDownloadedCount(0);
        setFailedCount(0);
        setSkippedCount(0);
        setShowStatus(false);

        const albumsToDownload = albums.filter(a => selectedAlbums.has(a.url));

        for (const album of albumsToDownload) {
            try {
                await DownloadAlbum(album.url, folder, "flac"); // Default to FLAC for now
            } catch (err) {
                // Error handled by event
            }
        }

        setIsDownloading(false);
        setShowStatus(true);
        addLog("All downloads finished", 'success');
    };

    return (
        <div className="flex h-screen bg-background text-white overflow-hidden">
            <Sidebar
                url={url}
                setUrl={setUrl}
                folder={folder}
                onSelectFolder={handleSelectFolder}
                onScan={handleScan}
                onStop={handleStopScan}
                isScanning={isScanning}
            />

            <main className="flex-1 flex flex-col h-full relative">
                {/* Header / Status */}
                <div className="p-8 pb-4">
                    <StatusPanel
                        show={showStatus}
                        downloadedCount={downloadedCount}
                        failedCount={failedCount}
                        skippedCount={skippedCount}
                    />

                    <div className="flex justify-between items-center mb-6">
                        <h2 className="text-xl font-bold">
                            Albums <span className="text-slate-500 text-sm font-normal ml-2">({albums.length})</span>
                        </h2>

                        {albums.length > 0 && (
                            <div className="flex space-x-4">
                                <button
                                    onClick={() => {
                                        const allFree = albums.filter(a => a.status !== 'paid').map(a => a.url);
                                        setSelectedAlbums(new Set(allFree));
                                    }}
                                    className="text-sm text-slate-400 hover:text-white transition-colors"
                                >
                                    Select All Free
                                </button>
                                <button
                                    onClick={handleDownload}
                                    disabled={isDownloading || selectedAlbums.size === 0}
                                    className="bg-emerald-500 hover:bg-emerald-600 disabled:opacity-50 disabled:cursor-not-allowed text-white px-6 py-2 rounded-lg font-medium transition-all shadow-lg shadow-emerald-500/20"
                                >
                                    {isDownloading ? 'Downloading...' : `Download Selected (${selectedAlbums.size})`}
                                </button>
                            </div>
                        )}
                    </div>
                </div>

                {/* Grid */}
                <div className="flex-1 overflow-y-auto px-8 pb-4">
                    {albums.length === 0 && !isScanning ? (
                        <div className="h-full flex flex-col items-center justify-center text-slate-600">
                            <p>Enter an artist URL and click Scan to start</p>
                        </div>
                    ) : (
                        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
                            {albums.map((album) => (
                                <AlbumCard
                                    key={album.url}
                                    album={album}
                                    isSelected={selectedAlbums.has(album.url)}
                                    onToggle={() => handleToggleAlbum(album.url)}
                                />
                            ))}
                        </div>
                    )}
                </div>

                {/* Logs */}
                <div className="p-8 pt-4 border-t border-slate-800 bg-surface/50 backdrop-blur-sm">
                    <LogPanel logs={logs} />
                </div>
            </main>
        </div>
    );
}

export default App;
