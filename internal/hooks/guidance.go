package hooks

// routingBlock is the static routing instructions injected into the system
// prompt at session start. It tells the LLM how memex-managed knowledge
// retrieval works and when to consult the KB versus answer directly.
//
// The TS reference (`hooks/core/routing.mjs`) ships this as plain Markdown;
// we mirror the wording so cross-platform behaviour stays identical until a
// later change replaces it with a per-session, KB-aware composition.
const routingBlock = `# ctx routing

You have access to a knowledge-base context layer managed by ctx.

- Prefer searching the KB before answering questions about this repo.
- Cite KB chunk IDs when you rely on retrieved context.
- If the KB returns nothing relevant, fall back to your own reasoning and say so.
`

// RoutingBlock returns the routing instructions injected into every session's
// system prompt. The string is static for now; later changes may parameterise
// it per platform or per session.
func RoutingBlock() string {
	return routingBlock
}
