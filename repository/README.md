# YLVEN Repository Layout

This directory is the machine-readable boundary for the YLVEN modular monorepo.
Each runtime module owns its public contracts and may not write another module's
tables directly. Cross-module changes use the API or event contracts under
`contracts/`.

The current work packet implements the repository and configuration boundaries
and the Android shell. Backend services and web applications are introduced by
their assigned P00 work packets.
