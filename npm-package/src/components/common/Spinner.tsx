import React from 'react';
import { Text } from 'ink';
import { useAnimation } from '../../contexts/AnimationContext.js';

const SPINNER_FRAMES = {
    dots: ['⠋', '⠙', '⠹', '⠸', '⠼', '⠴', '⠦', '⠧', '⠇', '⠏'],
};

interface SpinnerProps {
    type?: keyof typeof SPINNER_FRAMES;
    color?: string;
}

export const Spinner: React.FC<SpinnerProps> = ({ type = 'dots', color = 'yellow' }) => {
    const { frameIndex } = useAnimation();
    const frames = SPINNER_FRAMES[type];
    const frame = frames[frameIndex % frames.length];

    return <Text color={color}>{frame}</Text>;
};
