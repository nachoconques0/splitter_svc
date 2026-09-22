# Person is its own aggregate, not a field on a Share

The brief only ever shows names attached to a bill, so the obvious model is a
Share holding a name string. We modelled Person as a separate aggregate with its
own identity instead, because the same Person is expected to appear on several
Bills over time, and a name copied onto each Bill cannot express that.

## Consequences

Shares reference a Person by id, which means people have to be creatable and
discoverable before a Share can name one: hence `POST /people` and `GET /people`,
neither of which the brief asked for.

Removing someone from their last Bill leaves the Person in place. That is the
model working as intended, not a leak: if a Bill's lifecycle governed a Person's,
Person would be a value object with extra steps. Nothing garbage-collects them.

Rename and delete are deliberately absent. The brief's UI edits percentages, not
names, so a name is stable for as long as we need it to be.
