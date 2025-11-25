import React, { useEffect, useRef } from 'react';
import { LogMessage } from '../types';
import clsx from 'clsx';

interface LogPanelProps {
    logs: LogMessage[];
}

export const LogPanel: React.FC<LogPanelProps> = ({ logs }) => {
    const bottomRef = useRef<HTMLDivElement>(null);

    useEffect(() => {
        bottomRef.current?.scrollIntoView({ behavior: 'smooth' });
    }, [logs]);

    return (
        <div className="bg-black/50 rounded-lg p-4 h-48 overflow-y-auto font-mono text-xs border border-slate-800">
            {logs.length === 0 && (
                <div className="text-slate-600 italic">Waiting for activity...</div>
            )}
            {logs.map((log, i) => (
                <div key={i} className="mb-1 flex">
                    <span className="text-slate-500 mr-3">[{log.timestamp}]</span>
                    <span className={clsx(
                        log.type === 'error' ? 'text-red-400' :
                            log.type === 'success' ? 'text-emerald-400' :
                                log.type === 'warning' ? 'text-amber-400' :
                                    'text-slate-300'
                    )}>
                        {log.message}
                    </span>
                </div>
            ))}
            <div ref={bottomRef} />
        </div>
    );
};
