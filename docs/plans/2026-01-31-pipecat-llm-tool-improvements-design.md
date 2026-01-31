# Pipecat LLM Tool Improvements Design

**Date:** 2026-01-31
**Status:** Proposed
**Problem:** LLM doesn't call tool functions when it should - responds verbally instead of taking action

## Problem Statement

The pipecat-manager has 9 LLM tool functions for voice AI conversations. Currently, the LLM often fails to invoke these tools when users request actions, instead responding conversationally.

**Root Causes:**
1. Tool descriptions explain *what* they do but not *when* to use them
2. System prompt says "use tools when necessary" without concrete guidance
3. No trigger phrase examples or decision criteria
4. Similar tools (stop_service vs stop_flow) cause confusion

## Solution Overview

A two-layer guidance approach:

1. **Tool-Level (tools.py)** - Enhanced descriptions with:
   - Trigger phrases (user language patterns)
   - Decision criteria (when to use vs not use)
   - Disambiguation from similar tools
   - Concrete examples

2. **System-Level (main.go)** - Enhanced prompt with:
   - Action-first mindset instructions
   - Tool quick reference table
   - Example dialogues
   - Common mistakes to avoid

## File Changes

| File | Type |
|------|------|
| `bin-pipecat-manager/scripts/pipecat/tools.py` | Modify |
| `bin-ai-manager/pkg/aicallhandler/main.go` | Modify |

---

## Detailed Changes: tools.py

### Tool 1: CONNECT_CALL

```python
{
    "type": "function",
    "function": {
        "name": "connect_call",
        "description": """Connects the caller to another endpoint (person, department, or phone number).

WHEN TO USE:
- User asks to be transferred: "transfer me to...", "connect me to...", "put me through to..."
- User wants to speak to a person: "let me talk to a human", "I need an agent", "get me a representative"
- User requests a specific department: "sales", "support", "billing", "customer service"
- User provides a phone number to call: "call +1234567890", "dial my wife"

WHEN NOT TO USE:
- User mentions a person/department without requesting transfer (just discussing)
- User asks ABOUT a department but doesn't want to be connected
- User is asking for information you can provide directly

EXAMPLES:
- "Transfer me to sales" -> type="extension", target="sales"
- "Can you call my wife at 555-1234?" -> type="tel", target="+15551234"
- "I need to speak to a human" -> type="agent", target appropriate agent
- "Put me through to billing" -> type="extension", target="billing"

run_llm PARAMETER:
- true: Say something after connecting ("Connecting you now, please hold")
- false: Transfer silently or when call ends immediately after""",
        "parameters": { ... }  # Keep existing parameters
    }
}
```

### Tool 2: SEND_MESSAGE (SMS)

```python
{
    "type": "function",
    "function": {
        "name": "send_message",
        "description": """Sends an SMS text message to a phone number.

WHEN TO USE:
- User explicitly requests a text/SMS: "text me", "send me a text", "SMS me", "message my phone"
- User asks for information sent to their phone number
- User provides a phone number and asks for a message there

WHEN NOT TO USE:
- User says "message me" generically without specifying SMS (ask: email or text?)
- User wants an email (use send_email instead)
- User is discussing messaging but not requesting action

EXAMPLES:
- "Text me the confirmation number" -> send SMS with confirmation
- "Send an SMS to +1555123456 saying I'll be late" -> send that content
- "Can you message me the details?" -> ASK FIRST: "Would you like that as a text message or email?"

run_llm PARAMETER:
- true: Confirm verbally ("I've texted you the details")
- false: Send silently in background""",
        "parameters": { ... }  # Keep existing parameters
    }
}
```

### Tool 3: SEND_EMAIL

```python
{
    "type": "function",
    "function": {
        "name": "send_email",
        "description": """Sends an email to an email address.

WHEN TO USE:
- User explicitly requests email: "email me", "send me an email", "send that to my email"
- User asks for documents/information to be emailed
- User provides an email address for receiving information

WHEN NOT TO USE:
- User says "send me" or "message me" without specifying email (clarify first)
- User wants a text/SMS (use send_message instead)

EXAMPLES:
- "Email me the transcript" -> send email with transcript
- "Send the receipt to john@example.com" -> send to that address
- "Can you send me that?" -> ASK FIRST: "Would you like that by email or text message?"

run_llm PARAMETER:
- true: Confirm verbally ("I've sent that to your email")
- false: Send silently""",
        "parameters": { ... }  # Keep existing parameters
    }
}
```

### Tool 4: STOP_SERVICE

```python
{
    "type": "function",
    "function": {
        "name": "stop_service",
        "description": """Ends the AI conversation and proceeds to the next action in the flow.

WHEN TO USE:
- User says goodbye and conversation is complete: "bye", "goodbye", "thanks, that's all"
- User indicates they're done: "I'm all set", "that's everything", "nothing else"
- AI has successfully completed its purpose (appointment booked, issue resolved)
- Natural conversation conclusion

WHEN NOT TO USE:
- User is frustrated but still needs help (de-escalate instead)
- Conversation has unresolved issues
- User wants to END THE ENTIRE CALL (use stop_flow instead)

DIFFERS FROM stop_flow:
- stop_service = End AI portion, flow continues to next action
- stop_flow = Terminate everything immediately, no further actions

EXAMPLES:
- "Thanks, bye!" -> stop_service (natural end)
- "I'm done here" -> stop_service (user signals completion)
- After successfully booking appointment -> stop_service (task complete)
- "Great, that's all I needed" -> stop_service""",
        "parameters": { ... }  # Keep existing parameters
    }
}
```

### Tool 5: STOP_FLOW

```python
{
    "type": "function",
    "function": {
        "name": "stop_flow",
        "description": """Immediately terminates the entire flow/call. Nothing else executes after this.

WHEN TO USE:
- User explicitly wants to end everything: "hang up", "end the call", "terminate this"
- Critical error requiring full termination
- Emergency stop needed

WHEN NOT TO USE:
- User just wants to end AI conversation (use stop_service instead)
- User says casual goodbye (use stop_service instead)
- There are more flow actions that should execute after AI

DIFFERS FROM stop_service:
- stop_flow = HARD STOP - terminates everything, no further actions run
- stop_service = SOFT STOP - ends AI, flow continues normally

EXAMPLES:
- "Hang up now" -> stop_flow
- "End this call immediately" -> stop_flow
- "Just disconnect" -> stop_flow""",
        "parameters": { ... }  # Keep existing parameters
    }
}
```

### Tool 6: STOP_MEDIA

```python
{
    "type": "function",
    "function": {
        "name": "stop_media",
        "description": """Stops any audio/media currently playing.

WHEN TO USE:
- User asks to stop audio: "stop", "quiet", "be quiet", "shut up", "enough"
- User interrupts during long playback or hold music
- User wants immediate silence during media

WHEN NOT TO USE:
- User wants to end conversation (use stop_service)
- User wants to hang up (use stop_flow)
- No media is currently playing

EXAMPLES:
- "Stop talking" (during long TTS) -> stop_media
- "Be quiet" -> stop_media
- "OK I get it" (interrupting explanation) -> stop_media

run_llm PARAMETER:
- true: Respond after stopping ("OK, I'll stop. How can I help?")
- false: Just be silent""",
        "parameters": { ... }  # Keep existing parameters
    }
}
```

### Tool 7: SET_VARIABLES

```python
{
    "type": "function",
    "function": {
        "name": "set_variables",
        "description": """Saves key-value data to the flow context for later use.

WHEN TO USE (internal):
- Save information for downstream flow actions
- User provides data needed later: name, account number, preferences, choices
- Conversation reaches conclusions to record: appointment time, issue category, resolution

WHEN NOT TO USE:
- Information only needed for current response
- Data already stored elsewhere

EXAMPLES:
- User says "My name is John Smith" -> set_variables({"customer_name": "John Smith"})
- User confirms "3pm works" -> set_variables({"appointment_time": "15:00"})
- AI categorizes issue -> set_variables({"issue_category": "billing"})

run_llm PARAMETER:
- true: Continue conversation after saving
- false: Save silently while performing other actions""",
        "parameters": { ... }  # Keep existing parameters
    }
}
```

### Tool 8: GET_VARIABLES

```python
{
    "type": "function",
    "function": {
        "name": "get_variables",
        "description": """Retrieves previously saved variables from the flow context.

WHEN TO USE (internal):
- Need context set earlier in the flow
- Need information from previous actions
- User asks about something that should be in context

WHEN NOT TO USE:
- Information is already in conversation history
- Guessing if data exists (just try to retrieve it)

EXAMPLES:
- Need customer name collected earlier -> get_variables
- Previous action saved confirmation number -> get_variables
- User asks "what was my confirmation?" -> get_variables

run_llm PARAMETER:
- true: Respond using the retrieved data
- false: Retrieve silently before another action""",
        "parameters": { ... }  # Keep existing parameters
    }
}
```

### Tool 9: GET_AICALL_MESSAGES

```python
{
    "type": "function",
    "function": {
        "name": "get_aicall_messages",
        "description": """Retrieves message history from a specific AI call session.

WHEN TO USE (internal):
- Need message history from a different AI call
- Building summaries of past conversations
- User asks about previous interactions

WHEN NOT TO USE:
- Current conversation history is sufficient (already in context)
- Need variables, not messages (use get_variables)

EXAMPLES:
- User: "What did we discuss last time?" -> get_aicall_messages
- Generating summary of previous call -> get_aicall_messages

run_llm PARAMETER:
- true: Respond based on retrieved messages
- false: Retrieve for internal processing""",
        "parameters": { ... }  # Keep existing parameters
    }
}
```

---

## Detailed Changes: main.go

Add the following sections to `defaultCommonAIcallSystemPrompt`:

### Section 1: Action-First Mindset

```go
const defaultCommonAIcallSystemPrompt = `
// ... existing content ...

## CRITICAL: Action Over Talk

When a user requests an action you CAN perform with a tool, you MUST use the tool immediately.
DO NOT just describe what you could do - actually do it.

WRONG BEHAVIOR:
- User: "Transfer me to sales"
- AI: "I can transfer you to sales. Would you like me to do that?" (NO! Just do it!)

CORRECT BEHAVIOR:
- User: "Transfer me to sales"
- AI: "Connecting you to sales now." [invoke connect_call tool]

RULE: If user clearly requests an action and you have a tool for it, ACT IMMEDIATELY.
Only ask for clarification if genuinely missing required information.
`
```

### Section 2: Tool Quick Reference

```go
`
## Tool Quick Reference

| User Intent | Tool | Trigger Phrases |
|-------------|------|-----------------|
| Transfer/connect call | connect_call | "transfer me", "connect me to", "put me through", "let me speak to" |
| Send text message | send_message | "text me", "send SMS", "message my phone" |
| Send email | send_email | "email me", "send to my email" |
| End AI, continue flow | stop_service | "bye", "thanks that's all", "I'm done", "goodbye" |
| End entire call | stop_flow | "hang up", "end the call", "disconnect" |
| Stop audio playing | stop_media | "stop", "be quiet", "shut up" |
| Save data (internal) | set_variables | When user provides info to save for later |
| Get saved data (internal) | get_variables | When you need previously saved context |
| Get message history (internal) | get_aicall_messages | When referencing past conversation |
`
```

### Section 3: Example Dialogues

```go
`
## Example Dialogues

### Transfer Request
User: "I need to speak to someone in billing"
You: "I'll connect you to billing now."
Action: connect_call(destinations=[{type:"extension", target:"billing"}])

### SMS Request
User: "Text me the confirmation number"
Action: get_variables() to retrieve confirmation
You: "I'll text that to you now."
Action: send_message(text:"Your confirmation: ABC123", destinations=[caller's number])

### Ambiguous Request
User: "Send me the details"
You: "Would you prefer that by text message or email?"
[Wait for response, then use appropriate tool]

### Conversation End
User: "Great, thanks for your help!"
You: "You're welcome! Have a great day!"
Action: stop_service()

### Hang Up Request
User: "Just hang up"
Action: stop_flow()
`
```

### Section 4: Common Mistakes

```go
`
## Mistakes to Avoid

1. TALKING INSTEAD OF ACTING
   Wrong: "I can transfer you to sales if you'd like"
   Right: "Transferring you to sales now" + connect_call

2. UNNECESSARY CONFIRMATION
   Wrong: "Would you like me to send that text message?"
   Right: Just send it if user clearly requested it

3. CONFUSING STOP TOOLS
   - stop_media = Stop audio playback only
   - stop_service = End AI, flow continues to next action
   - stop_flow = End everything, call terminates

4. FORGETTING TO ACT
   If user says "transfer me to sales", you MUST call connect_call.
   Acknowledging without acting is WRONG.
`
```

---

## Testing Plan

After implementation, test these scenarios:

| Test Case | User Input | Expected Behavior |
|-----------|------------|-------------------|
| Direct transfer | "Transfer me to sales" | connect_call invoked immediately |
| Human request | "Let me speak to a human" | connect_call to agent |
| SMS request | "Text me the details" | send_message invoked |
| Email request | "Email me the transcript" | send_email invoked |
| Ambiguous send | "Send me that info" | Asks: email or text? |
| Goodbye | "Thanks, bye!" | stop_service invoked |
| Hang up | "Hang up now" | stop_flow invoked |
| Stop audio | "Be quiet" | stop_media invoked |
| Stop confusion | "I'm done, goodbye" | stop_service (not stop_flow) |

---

## Implementation Steps

1. Update `bin-pipecat-manager/scripts/pipecat/tools.py`
   - Replace each tool's description with enhanced version
   - Keep parameter definitions unchanged

2. Update `bin-ai-manager/pkg/aicallhandler/main.go`
   - Append new sections to `defaultCommonAIcallSystemPrompt`
   - Keep existing prompt content

3. Test with real LLM calls
   - Verify tool invocation on trigger phrases
   - Verify clarification on ambiguous requests
   - Verify correct tool selection (stop_service vs stop_flow)

4. Deploy and monitor
   - Watch for tool invocation rates
   - Collect cases where LLM still fails to invoke tools
