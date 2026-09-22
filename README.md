# Bill splitter

Split a bill between people by percentage. Go API, Vue 3 frontend, Postgres.

## Run it

```sh
make up
```

Open <http://localhost:8080>. There's already a bill seeded so you can use it
straight away.

`make down` stops everything, `make clean` also drops the volumes. `make test`
runs the tests and doesn't need a database.

```
GET  /bills/{id}/shares   the bill, its people and their percentages
PUT  /bills/{id}/shares   replace the whole set of shares
POST /people              create a person
GET  /people              list people
```

## Decisions

**Whole numbers everywhere.** 33.33% is stored as 3333, and €85.00 as 8500
cents. Computers can't hold decimals exactly, so adding them up can come out
slightly wrong. Whole numbers can't. Checking a bill is fully split is then just
3333 + 3333 + 3334 = 10000.

**Interfaces are declared where they're used.** The service defines the
repository interface it needs, and the controller defines the service interface
it needs. There's no shared package of contracts that everything imports. It
also keeps the interfaces small, so the generated mocks only carry the methods
the caller actually uses.

**One test seam, at the HTTP boundary.** The tests run the real router,
controller and service, and mock only the repository. They check status codes
and response bodies rather than which methods were called, so moving things
around inside doesn't break them.

**Transactions stay in the repository.** No `*sql.Tx` crosses a layer boundary.
The service asks for the shares to be replaced, and the repository decides that
means a delete and some inserts in one transaction.

## What I'd change with more time

- Tests against a real Postgres. The repository is mocked, so I checked the
  rollback and the version guard by hand.
- Currencies that don't use two decimal places.
- Component tests for the page. Only the allocation is tested right now.
- Domain events, so something can react when a bill is settled.

## Left out on purpose

Auth, accounts, visual design, CI, deployment and more than one bill in the UI,
which the brief excludes.

I also skipped renaming and deleting people, search on the people list, and
merging two conflicting edits. On a conflict you get a reload button and your
typing stays put.
