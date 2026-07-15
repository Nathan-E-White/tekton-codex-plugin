# Execute mutations through single-use plans

Every cluster mutation is represented by an immutable 15-minute Mutation Plan and requires its plan ID plus an exact confirmation token. The extra round trip prevents stale, replayed, context-switched, or model-invented writes even when the MCP client also provides approval prompts.
