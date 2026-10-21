import { useState, useEffect } from 'react';
import { Sidebar } from './components/Sidebar';
import { AlbumCard } from './components/AlbumCard';
import { LogPanel } from './components/LogPanel';
import { StatusPanel } from './components/StatusPanel';
import { Album, LogMessage } from './types';

const api = {
    scan: (url: string) => fetch('/api/scan', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ url })
    }),
    stopScan: () => fetch('/api/scan/stop', { method: 'POST' }),
    download: (url: string, format: string) => fetch('/api/download', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ url, format })
    }),
    getConfig: () => fetch('/api/config').then(r => r.json()),
};

function App() {
    const [url, setUrl] = useState("");
    const [folder, setFolder] = useState("/downloads");
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
        // Fetch server config
        api.getConfig().then(cfg => {
            if (cfg.download_dir) setFolder(cfg.download_dir);
        }).catch(() => setFolder('/downloads'));

        // Server-Sent Events listener
        const es = new EventSource('/api/events');

        es.addEventListener('connected', () => {
            console.log('Connected to SSE event stream');
        });

        es.addEventListener('scan:start', (e: MessageEvent) => {
            setIsScanning(true);
            addLog(`Scanning artist: ${e.data}`, 'info');
        });

        es.addEventListener('scan:complete', (e: MessageEvent) => {
            setIsScanning(false);
            try {
                const results: Album[] = JSON.parse(e.data);
                setAlbums(results || []);
                addLog(`Found ${results?.length || 0} albums`, 'success');
            } catch (err) {
                addLog('Error parsing scan results', 'error');
            }
        });

        es.addEventListener('scan:album_found', (e: MessageEvent) => {
            try {
                const album: Album = JSON.parse(e.data);
                addLog(`Found album: ${album.title}`, 'info');
                setAlbums(prev => {
                    if (prev.some(a => a.url === album.url)) return prev;
                    return [...prev, album];
                });
            } catch (err) {
                console.error('Error parsing album:', err);
            }
        });

        es.addEventListener('scan:error', (e: MessageEvent) => {
            setIsScanning(false);
            addLog(`Scan error: ${e.data}`, 'error');
        });

        es.addEventListener('scan:stopped', (e: MessageEvent) => {
            setIsScanning(false);
            addLog(`Scan stopped. Found ${e.data} albums.`, 'warning');
        });

        es.addEventListener('download:start', (e: MessageEvent) => {
            addLog(`Starting download: ${e.data}`, 'info');
        });

        es.addEventListener('download:progress', (e: MessageEvent) => {
            try {
                const data = JSON.parse(e.data);
                addLog(data.message, 'info');
            } catch (err) {
                addLog(e.data, 'info');
            }
        });

        es.addEventListener('download:complete', (e: MessageEvent) => {
            setDownloadedCount(prev => prev + 1);
            addLog(`Download complete: ${e.data}`, 'success');
        });

        es.addEventListener('download:error', (e: MessageEvent) => {
            setFailedCount(prev => prev + 1);
            try {
                const data = JSON.parse(e.data);
                addLog(`Download failed: ${data.error}`, 'error');
            } catch (err) {
                addLog(`Download failed: ${e.data}`, 'error');
            }
        });

        return () => {
            es.close();
        };
    }, []);

    const handleScan = async () => {
        if (!url) return;
        setAlbums([]);
        setSelectedAlbums(new Set());
        setShowStatus(false);
        setIsScanning(true);

        try {
            await api.scan(url);
        } catch (err) {
            console.error('Scan error:', err);
            setIsScanning(false);
            addLog(`Failed to start scan: ${err}`, 'error');
        }
    };

    const handleStopScan = async () => {
        try {
            await api.stopScan();
        } catch (err) {
            console.error('Stop scan error:', err);
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
                await api.download(album.url, "flac");
            } catch (err) {
                // Handled by SSE events
            }
        }

        setIsDownloading(false);
        setShowStatus(true);
        addLog("All downloads queued", 'success');
    };

    return (
        <div className="flex h-screen bg-background text-white overflow-hidden">
            <Sidebar
                url={url}
                setUrl={setUrl}
                folder={folder}
                onSelectFolder={() => {}}
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
