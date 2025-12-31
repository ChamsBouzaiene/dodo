import React, { useState } from 'react';
import { Box, Text, useApp, useInput } from 'ink';
import { Input as TextInput } from './common/Input.js'; // Assuming we have a reusable Input component
import { useEngineConnection } from '../hooks/useEngineConnection.js';
import { debugLog } from '../utils/debugLogger.js';


type SetupStep = 'intro' | 'provider' | 'api_key' | 'model' | 'embedding_key' | 'auto_index' | 'summary' | 'saving' | 'complete';

const PROVIDERS = ['openai', 'anthropic', 'kimi', 'gemini', 'deepseek', 'groq', 'lmstudio', 'ollama', 'glm', 'minimax'];

export type SetupConfig = {
    llm_provider: string;
    api_key: string;
    model: string;
    auto_index: boolean;
    embedding_key: string;
};

/**
 * Props for the SetupWizard component.
 */
export interface SetupWizardProps {
    /** Callback to send command to the engine */
    sendCommand: (type: string, payload: Record<string, string | number | boolean>) => void;
    /** Callback when setup is complete */
    onComplete: () => void;
    /** Whether the wizard is in update mode */
    isUpdate?: boolean;
    /** Initial configuration to populate fields */
    initialConfig?: Record<string, string>;
}


// Helper component for uncontrolled input behavior
const UncontrolledInput: React.FC<{ placeholder?: string, onSubmit: (val: string) => void }> = ({ placeholder, onSubmit }) => {
    const [value, setValue] = useState("");
    return (
        <TextInput
            value={value}
            onChange={setValue}
            onSubmit={onSubmit}
            placeholder={placeholder}
        />
    );
};

const PROVIDER_MODELS: Record<string, string[]> = {
    'openai': ['gpt-4o', 'gpt-4o-mini', 'gpt-4-turbo'],
    'anthropic': ['claude-3-5-sonnet-20240620', 'claude-3-opus-20240229', 'claude-3-sonnet-20240229', 'claude-3-haiku-20240307'],
    'kimi': ['kimi-k2-250711'],
};

export const SetupWizard: React.FC<SetupWizardProps> = ({ sendCommand, onComplete, isUpdate = false, initialConfig }) => {
    const { exit } = useApp();
    const [step, setStep] = useState<SetupStep>('intro');
    const [isReady, setIsReady] = useState(false);

    // Initialize config with defaults or initialConfig
    const [config, setConfig] = useState({
        llm_provider: 'openai',
        api_key: '',
        model: 'gpt-4o',
        auto_index: true,
        embedding_key: '',
    });

    // Populate config from props when available
    React.useEffect(() => {
        if (initialConfig) {
            setConfig(prev => ({
                ...prev,
                llm_provider: initialConfig.llm_provider || prev.llm_provider,
                api_key: initialConfig.api_key || prev.api_key,
                model: initialConfig.model || prev.model,
                auto_index: initialConfig.auto_index === 'true'
            }));
        }
    }, [initialConfig]);

    const [selectedProviderIndex, setSelectedProviderIndex] = useState(0);
    const [selectedModelIndex, setSelectedModelIndex] = useState(0);

    // Track last step change to prevent rapid skipping
    const lastStepChangeRef = React.useRef(Date.now());

    // Prevent immediate keypress handling
    React.useEffect(() => {
        debugLog.lifecycle('SetupWizard', 'mount', `isUpdate=${isUpdate}`);
        const timer = setTimeout(() => setIsReady(true), 1000);
        return () => {
            debugLog.lifecycle('SetupWizard', 'unmount');
            clearTimeout(timer);
        };
    }, [isUpdate]);

    useInput((input, key) => {
        // Debug: Log all keypresses with current step
        debugLog.event('SetupWizard', 'keypress', { step, input, key });

        if (!isReady) return;

        // Force 500ms cooldown between steps
        if (Date.now() - lastStepChangeRef.current < 500) {
            return;
        }

        if (key.ctrl && input === 'c') {
            exit();
            return;
        }

        if (step === 'intro' && key.return) {
            setStep('provider');
            lastStepChangeRef.current = Date.now();
            return;
        }

        if (step === 'provider') {
            if (key.upArrow) {
                setSelectedProviderIndex(prev => Math.max(0, prev - 1));
            }
            if (key.downArrow) {
                setSelectedProviderIndex(prev => Math.min(PROVIDERS.length - 1, prev + 1));
            }
            if (key.return) {
                const provider = PROVIDERS[selectedProviderIndex];
                setConfig(prev => ({ ...prev, llm_provider: provider, model: PROVIDER_MODELS[provider][0] }));
                setSelectedModelIndex(0); // Reset model selection
                setStep('model');
                lastStepChangeRef.current = Date.now();
            }
            return;
        }

        if (step === 'model') {
            const models = PROVIDER_MODELS[config.llm_provider] || [];
            if (key.upArrow) {
                setSelectedModelIndex(prev => Math.max(0, prev - 1));
            }
            if (key.downArrow) {
                setSelectedModelIndex(prev => Math.min(models.length - 1, prev + 1));
            }
            if (key.return) {
                setConfig(prev => ({ ...prev, model: models[selectedModelIndex] }));
                setStep('api_key');
                lastStepChangeRef.current = Date.now();
            }
            return;
        }

        if (step === 'auto_index') {
            if (input === 'y' || input === 'Y' || key.return) {
                setConfig(prev => ({ ...prev, auto_index: true }));
                setStep('summary');
                lastStepChangeRef.current = Date.now();
                return;
            }
            if (input === 'n' || input === 'N') {
                setConfig(prev => ({ ...prev, auto_index: false }));
                setStep('summary');
                lastStepChangeRef.current = Date.now();
                return;
            }
        }

        // Summary step navigation
        if (step === 'summary') {
            if (key.return) {
                handleSave();
                lastStepChangeRef.current = Date.now();
                return;
            }
            if (key.escape || key.backspace || key.delete) {
                setStep('intro');
                return;
            }
        }

        // Final enter to launch
        if (step === 'complete' && key.return) {
            debugLog.lifecycle('SetupWizard', 'update', 'onComplete triggered');
            onComplete();
        }
    }, { isActive: true });

    const handleSave = () => {
        setStep('saving');

        // Convert config to string map for backend protocol compatibility
        const payload = {
            ...config,
            auto_index: config.auto_index ? "true" : "false"
        };

        sendCommand('save_config', payload);

        // Artificial delay for UX
        setTimeout(() => {
            setStep('complete');
        }, 1000);
    };

    if (step === 'intro') {
        return (
            <Box flexDirection="column" padding={1} borderStyle="round" borderColor="cyan">
                <Text bold color="cyan">{isUpdate ? "Update Configuration" : "Welcome to Dodo! 🦤"}</Text>
                <Text>{isUpdate ? "Let's update your Dodo agent settings." : "It looks like this is your first time running Dodo."}</Text>
                {!isUpdate && <Text>Let's get you set up in a few seconds.</Text>}

                {isUpdate && (
                    <Box marginTop={1} flexDirection="column">
                        <Text color="gray">You are currently re-configuring the agent.</Text>
                    </Box>
                )}

                <Box marginTop={1}>
                    <Text color="green">Press Enter to start...</Text>
                </Box>
            </Box>
        );
    }

    if (step === 'provider') {
        return (
            <Box flexDirection="column" padding={1} borderStyle="round" borderColor="blue">
                <Text bold>Select your LLM Provider:</Text>
                {PROVIDERS.map((p, i) => (
                    <Text key={p} color={i === selectedProviderIndex ? 'green' : 'white'}>
                        {i === selectedProviderIndex ? '> ' : '  '} {p}
                    </Text>
                ))}
                <Box marginTop={1}>
                    <Text color="gray">Use Up/Down arrows, Enter to select</Text>
                </Box>
            </Box>
        )
    }

    if (step === 'model') {
        const models = PROVIDER_MODELS[config.llm_provider] || [];
        return (
            <Box flexDirection="column" padding={1} borderStyle="round" borderColor="blue">
                <Text bold>Select Default Model for {config.llm_provider}:</Text>
                {models.map((m, i) => (
                    <Text key={m} color={i === selectedModelIndex ? 'green' : 'white'}>
                        {i === selectedModelIndex ? '> ' : '  '} {m}
                    </Text>
                ))}
                <Box marginTop={1}>
                    <Text color="gray">Use Up/Down arrows, Enter to select</Text>
                </Box>
            </Box>
        );
    }

    if (step === 'api_key') {
        return (
            <Box flexDirection="column" padding={1} borderStyle="round" borderColor="blue">
                <Text bold>Enter your API Key for {config.llm_provider}:</Text>
                <UncontrolledInput
                    key={step}
                    placeholder="sk-..."
                    onSubmit={(val) => {
                        if (!val.trim()) return; // Prevent empty submission
                        setConfig(prev => ({ ...prev, api_key: val }));
                        // If OpenAI, skip embedding_key step (reuse same key)
                        if (config.llm_provider === 'openai') {
                            setStep('auto_index');
                        } else {
                            setStep('embedding_key');
                        }
                    }}
                />
            </Box>
        );
    }

    if (step === 'embedding_key') {
        return (
            <Box flexDirection="column" padding={1} borderStyle="round" borderColor="blue">
                <Text bold>Optional: Enter OpenAI API Key for Embeddings</Text>
                <Text color="gray">Semantic search requires OpenAI embeddings. Press Enter to skip.</Text>
                <UncontrolledInput
                    key={step}
                    placeholder="sk-... (optional)"
                    onSubmit={(val) => {
                        setConfig(prev => ({ ...prev, embedding_key: val.trim() }));
                        setStep('auto_index');
                    }}
                />
            </Box>
        );
    }

    if (step === 'auto_index') {
        return (
            <Box flexDirection="column" padding={1} borderStyle="round" borderColor="blue">
                <Text bold>Enable Auto-Indexing for new projects? (Y/n)</Text>
                <Text color="gray">This allows Dodo to understand your codebase using semantic search.</Text>
            </Box>
        );
    }

    if (step === 'summary') {
        return (
            <Box flexDirection="column" padding={1} borderStyle="round" borderColor="yellow">
                <Text bold color="yellow">Configuration Summary</Text>
                <Box marginY={1} flexDirection="column">
                    <Text>Provider:   <Text bold color="white">{config.llm_provider}</Text></Text>
                    <Text>Model:      <Text bold color="white">{config.model}</Text></Text>
                    <Text>API Key:    <Text bold color="white">{config.api_key ? '********' + config.api_key.slice(-4) : '(Not set)'}</Text></Text>
                    {config.llm_provider !== 'openai' && (
                        <Text>Embed Key:  <Text bold color="white">{config.embedding_key ? '********' + config.embedding_key.slice(-4) : '(Not set - no semantic search)'}</Text></Text>
                    )}
                    <Text>Auto-Index: <Text bold color="white">{config.auto_index ? 'Yes' : 'No'}</Text></Text>
                </Box>
                <Text>Press <Text bold color="green">Enter</Text> to Save, or <Text bold color="red">Esc</Text> to Restart</Text>
            </Box>
        );
    }

    if (step === 'saving') {
        return (
            <Box flexDirection="column" padding={1}>
                <Text color="yellow">Saving configuration...</Text>
            </Box>
        );
    }

    if (step === 'complete') {
        return (
            <Box flexDirection="column" padding={1} borderStyle="round" borderColor="green">
                <Text bold color="green">Setup Complete! 🎉</Text>
                <Text>Configuration saved successfully.</Text>
                <Box marginTop={1}>
                    <Text color="gray">Press Enter to return...</Text>
                </Box>
            </Box>
        );
    }

    return null;
};
