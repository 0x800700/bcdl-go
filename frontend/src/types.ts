export interface Album {
    title: string;
    artist: string;
    coverUrl: string;
    url: string;
    isFree: boolean;
    isNyp: boolean;
    price: string;
    status: string; // "free", "nyp", "paid"
}

export interface LogMessage {
    timestamp: string;
    message: string;
    type: 'info' | 'success' | 'error' | 'warning';
}

export interface DownloadProgress {
    url: string;
    message: string;
}
