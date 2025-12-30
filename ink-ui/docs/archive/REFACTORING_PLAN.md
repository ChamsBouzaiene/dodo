# Ink UI Refactoring Plan

## Issues Found

### 1. **TypeScript Configuration** (Critical)
- `moduleResolution: "Node"` doesn't work with ES modules
- Should use `"node16"`, `"nodenext"`, or `"bundler"`

### 2. **Component Size** (High Priority)
- `app.tsx` is 447 lines - violates single responsibility
- Should split into:
  - `App.tsx` - Main orchestrator
  - `hooks/useEngineEvents.ts` - Event handling logic
  - `hooks/useSession.ts` - Session management
  - `components/Header.tsx`
  - `components/Conversation.tsx`
  - `components/Sidebar.tsx`
  - `components/Footer.tsx`

### 3. **State Management** (High Priority)
- 10+ `useState` calls - hard to track
- Consider `useReducer` for complex state
- Or extract to custom hooks

### 4. **Event Handler Dependencies** (Medium Priority)
- Large `useEffect` with many dependencies
- Event handlers recreated on every render
- Should use `useCallback` for handlers

### 5. **Memory Leaks** (Medium Priority)
- Event listeners might not clean up if component unmounts during async operations
- `readline` interface in `engineClient.ts` never explicitly closed

### 6. **Error Handling** (Medium Priority)
- No error boundaries
- Errors from event handlers could crash the app
- No retry logic for failed commands

### 7. **Race Conditions** (Low Priority)
- `mounted` flag pattern works but could use `AbortController` for better cancellation
- Multiple state updates could cause inconsistent UI

### 8. **Code Duplication** (Low Priority)
- Turn manipulation logic repeated
- Status color mapping duplicated
- Path utilities could be extracted

## Refactoring Steps

### Phase 1: Fix TypeScript Config
1. Update `tsconfig.json` moduleResolution
2. Fix import paths if needed

### Phase 2: Extract Custom Hooks
1. `useEngineEvents` - Handle all protocol events
2. `useSession` - Manage session lifecycle
3. `useConversation` - Manage turns/assistant text

### Phase 3: Split Components
1. Extract Header, Conversation, Sidebar, Footer
2. Create shared types file
3. Add proper prop types

### Phase 4: Improve State Management
1. Consider `useReducer` for app state
2. Or keep hooks but memoize properly

### Phase 5: Add Error Handling
1. Error boundaries (if Ink supports)
2. Better error messages
3. Retry logic for failed operations

### Phase 6: Performance
1. Memoize expensive computations
2. Use `useCallback` for event handlers
3. Optimize re-renders


