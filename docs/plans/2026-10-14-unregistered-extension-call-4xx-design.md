# VOIP-1593 Return 4xx for an unregistered extension destination

## 1. Ticket review (valid)
POST /calls with a type=extension destination that has no registered contact returns 500 INTERNAL. Verified on origin/main a6343d1fd:
- groupcallhandler/dial.go getDialDestinationsAddressTypeExtension returns a plain fmt.Errorf when the contact list is empty.
- Start -> startWithDestination -> getDialAddressesAndRingMethod wrap it with errors.Wrap (pkg/errors v0.9.1 keeps Unwrap).
- callhandler CreateCallsOutgoing logs each destination error and continues; when every destination failed it returns a fresh fmt.Errorf, dropping the cause.
- listenhandler errorResponse maps only a typed *VoipbinError to its status; a plain error becomes 500.
- api-manager already passes a typed error through (CallCreate -> abortWithServiceError -> translateToVoipbinError); no change needed there.
Partial failure (some destinations succeed) already succeeds and skips the failed ones; unchanged.

## 2. Design
- dial.go: return cerrors.FailedPrecondition(call-manager, "EXTENSION_NOT_REGISTERED", "The destination extension \"<target_name>\" is not currently registered.") (wire 409). The message echoes the caller-supplied extension name so a multi-destination request can tell which one failed; it carries no internal text. FAILED_PRECONDITION over INVALID_ARGUMENT: the request is well formed, the destination state is the problem.
- CreateCallsOutgoing: in the groupcall branch only, remember the first typed *VoipbinError; when all destinations failed return it, otherwise keep the existing plain error. The sip/tel branch is untouched, so no other 500 path changes status (out of scope). Mixed requests: the first typed error in destination order wins; a plain failure in another destination is ignored when all failed. A code comment records why only the groupcall branch preserves the typed error.

## 3. Reach (verified)
- Inside groupcall Start, a typed error is returned synchronously only for a single-destination ring_all start; multi-destination ring_all and linear fail inside goroutines (intentionally out of scope).
- Surfaces that now return 409: POST /calls (CreateCallsOutgoing) and POST /groupcalls (processV1GroupcallsPost -> errorResponse), plus the admin provider test call that wraps CallV1CallsCreate.
- Behavior by endpoint: POST /calls calls Start once per destination with a single destination, so it returns 409 when every destination failed (any extension destination unregistered, first typed error wins) and 200 when at least one destination succeeded. POST /groupcalls returns 409 only for a single-destination ring_all; multi-destination and linear start asynchronously and keep returning 200.
- Out of scope: typed errors raised in the sip/tel branch keep being dropped (generic 500); the empty target_name case echoes an empty name; the temporary flow created by POST /calls is not deleted on failure (existing behavior).
- Callers flow, queue, route, transfer, api managers only branch on err != nil. ai-manager create_call passes err.Error() to the LLM, so its text becomes the typed message (an improvement).

## 4. Verification
- RED/GREEN: Test_Start_extensionNotRegistered_typedError, Test_CreateCallsOutgoing_groupcallStartFailure (+ partial-success and sip-branch-unchanged cases) and a reason assertion in Test_getDialDestinationsAddressTypeExtension_noContacts and a listenhandler errorResponse 409 assertion. Before the PR: git fetch origin main and re-check the touched files for upstream changes, recorded on the card.
- Registered-contact path unchanged (existing tests).
- Docs: add '409' -> components/responses/Conflict to POST /calls and POST /groupcalls in bin-openapi-manager (regenerate, build bin-api-manager), and a short 409 EXTENSION_NOT_REGISTERED note in call_tutorial.rst and call_groupcall.rst, plus an EXTENSION_NOT_REGISTERED row (409) in the Call Reasons table of restful_api_errors.rst with a clean Sphinx rebuild; the POST /calls and POST /groupcalls wording follows the endpoint behavior above (build/ force-added per repo rule).
