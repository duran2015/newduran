# M0 game package fixtures

These fixtures deliberately contain no Speeding vocabulary. They exercise the
generic M0 package contract with two unrelated domains:

- `counter-v1.json`: one metric and three actions, for the smallest possible
  Command → Effect → Event → State Diff example.
- `restaurant-v1.json`: inventory, cash and operating status, plus purchase,
  wait, supply interruption and closure actions.
- `restaurant-morale-v1.1.json`: the same restaurant package upgraded only by
  manifest data. It adds `employee_morale`, a `team_building` action and a new
  event without adding a Kernel operation or changing Kernel code.

The v1.1 fixture repeats the v1 registry intentionally. At M0 a package version
is immutable and self-contained; patch semantics and package dependencies are
not part of the Kernel contract yet.
