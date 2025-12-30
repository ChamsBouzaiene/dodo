import React, { createContext, useContext, useState, useEffect, ReactNode } from 'react';

interface AnimationContextType {
    frameIndex: number;
}

const AnimationContext = createContext<AnimationContextType | undefined>(undefined);

export const useAnimation = () => {
    const context = useContext(AnimationContext);
    if (!context) {
        throw new Error('useAnimation must be used within an AnimationProvider');
    }
    return context;
};

interface AnimationProviderProps {
    children: ReactNode;
    interval?: number;
}

export const AnimationProvider: React.FC<AnimationProviderProps> = ({ children, interval = 80 }) => {
    const [frameIndex, setFrameIndex] = useState(0);

    useEffect(() => {
        const timer = setInterval(() => {
            setFrameIndex(prev => (prev + 1) % 1000); // Wrap around occasionally
        }, interval);

        return () => clearInterval(timer);
    }, [interval]);

    return (
        <AnimationContext.Provider value={{ frameIndex }}>
            {children}
        </AnimationContext.Provider>
    );
};
