# Bill splitter

A take-home exercise: split a Bill between people by percentage. Go API, Vue 3
frontend, Postgres.

## Read these first

- `CONTEXT.md` — the domain glossary. Use these words in code, comments, commits
  and tickets: Bill, Total, Person, Share, Share Set, Percentage, Amount,
  Allocation, Remainder. Do not introduce synonyms.
- `docs/adr/` — four decisions and why. Respect them; if one looks wrong, say so
  rather than quietly working around it.
- `.scratch/bill-splitting/spec.md` — the full spec.
- `.scratch/bill-splitting/issues/` — the tickets, in dependency order.

## Working rhythm

One ticket per context window. Read the ticket, build it end to end, run the
tests, commit, clear, take the next one whose blockers are done. Tickets 04, 05
and 06 are independent of each other and all unblock once 03 lands.

## Hard rules

- **No floats anywhere near a Percentage or an Amount.** Percentages are integer
  hundredths of a percent (100.00% is 10000); money is integer minor units plus a
  currency code. Parse inbound numbers as `json.Number`, digit by digit. The
  sum-to-100 rule is an exact integer comparison — there is no tolerance value in
  this codebase and none should appear.
- **Percentages and money cross the wire as decimal strings** (`"33.33"`,
  `"25.00"`). Basis points and minor units are internal.
- **Validate before opening a transaction.** A rejected Share Set must never
  touch the database.

## Backend conventions

Follows the house style of the author's other Go services:

- `cmd/server` plus `internal/{app,config,http,controller,service,repository,entity,model}`.
  One directory per domain under controller/service/repository/entity, with a
  fixed filename per layer; `model` is a flat package of DTOs.
- Two domains: one covering Bill and Share, one covering Person.
- **Interfaces are declared at the point of use**, not in a central ports
  package. The controller declares the Service interface it needs; the service
  declares the Repository interface it needs.
- gin for HTTP. `database/sql` with hand-written SQL — no ORM. Repository methods
  take `context.Context` first.
- Transactions are opened and committed **inside the repository**. No `*sql.Tx`
  crosses a layer boundary.
- Entities are framework-free structs. Sentinel errors are package vars in the
  entity package, matched with `errors.Is`, mapped to the error envelope at the
  service or controller boundary.
- Error responses carry a numeric code, a machine-readable `error_code` in
  **snake_case**, a message, and an optional detail. Unmapped errors default to
  500 so infrastructure messages never leak. Clients branch on `error_code`,
  never on message text.
- Config is environment variables only, with named default consts. Missing
  required variables are reported together and stop startup.
- `log/slog` for structured logging.
- Comments explain **why**, not what. This is the strongest signature of the
  author's codebases — a comment earning its place explains a decision, a failure
  mode, or a constraint that is not visible in the code.

## Frontend conventions

- Vue 3 + TypeScript + Vite + Quasar. `npm`. Build is `vue-tsc -b && vite build`,
  so type errors fail the build.
- SFC block order is `<template>`, then `<script setup lang="ts">`, then
  `<style scoped>`. PascalCase filenames. Type-only `defineProps`/`defineEmits`
  with `withDefaults`.
- An axios instance plus a route map of endpoint functions. Never raw `fetch`.
- Composables named `use<Noun>()` returning `{ loading, error, data, … }`, and
  returning a success-or-failure result rather than throwing. No Pinia — there is
  no cross-cutting state here.
- Errors map through a shared handler that reads `error_code` and displays the
  server's message.
- Quasar form rules for validation; no form library.
- Currency formatting lives in one utility. Do not duplicate it per component.

## Testing

- **One seam: the HTTP boundary.** Drive a real router with the real controller
  and service, and a mocked repository. Assert status, `error_code` and body.
  Never assert which repository methods were called in what order.
- Mocks are **generated with gomock** (`go.uber.org/mock`) from the interfaces,
  via generate directives. This departs from the author's other projects, which
  hand-write fakes; it is deliberate — see the spec's testing section.
- Tests live beside the code in external `_test` packages, table-driven with
  subtests, using `httptest` against a router in test mode.
- One frontend test only: the TypeScript Allocation, with the same cases as the
  Go table.
- `go test ./...` must pass without Docker running.

## Commands

The single start command, the migration targets and the test commands are
defined in the Makefile and `docker-compose.yml`. Read them rather than guessing;
if a command in this file disagrees with the Makefile, the Makefile is right.
