import React from 'react';
import { render } from 'ink-testing-library';
import { describe, it, expect, vi } from 'vitest';
import { Input } from '../../components/common/Input';


describe('Input Component', () => {
    it('renders initial value correctly', () => {
        const { lastFrame } = render(
            <Input value="test value" onChange={() => { }} onSubmit={() => { }} />
        );
        expect(lastFrame()).toContain('test value');
    });

    it('renders placeholder when empty', () => {
        const { lastFrame } = render(
            <Input value="" placeholder="Type here..." onChange={() => { }} onSubmit={() => { }} />
        );
        expect(lastFrame()).toContain('Type here...');
    });

    it('calls onChange when typing', async () => {
        const onChange = vi.fn();
        const { stdin } = render(
            <Input value="" onChange={onChange} onSubmit={() => { }} />
        );

        await new Promise(r => setTimeout(r, 10));
        stdin.write('a');
        await new Promise(r => setTimeout(r, 10));
        expect(onChange).toHaveBeenCalledWith('a');
    });

    it('calls onSubmit when pressing Enter', async () => {
        const onSubmit = vi.fn();
        const { stdin } = render(
            <Input value="submit me" onChange={() => { }} onSubmit={onSubmit} />
        );

        await new Promise(r => setTimeout(r, 10));
        stdin.write('\r'); // Return key
        await new Promise(r => setTimeout(r, 10));
        expect(onSubmit).toHaveBeenCalledWith('submit me');
    });

    it('calls onHistoryUp when pressing Up arrow', async () => {
        const onUp = vi.fn();
        const { stdin } = render(
            <Input value="" onChange={() => { }} onSubmit={() => { }} onHistoryUp={onUp} />
        );

        await new Promise(r => setTimeout(r, 10));
        stdin.write('\u001B[A'); // Up arrow ANSI code
        await new Promise(r => setTimeout(r, 10));
        expect(onUp).toHaveBeenCalled();
    });

    it('calls onHistoryDown when pressing Down arrow', async () => {
        const onDown = vi.fn();
        const { stdin } = render(
            <Input value="" onChange={() => { }} onSubmit={() => { }} onHistoryDown={onDown} />
        );

        await new Promise(r => setTimeout(r, 10));
        stdin.write('\u001B[B'); // Down arrow ANSI code
        await new Promise(r => setTimeout(r, 10));
        expect(onDown).toHaveBeenCalled();
    });

    it('does not emit events when disabled', () => {
        const onChange = vi.fn();
        const { stdin } = render(
            <Input value="" onChange={onChange} onSubmit={() => { }} isDisabled={true} />
        );

        stdin.write('a');
        expect(onChange).not.toHaveBeenCalled();
    });
});
