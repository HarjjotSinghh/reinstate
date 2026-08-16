// Package agents is the catalog of coding agents Reinstate knows about.
//
// One Descriptor is registered per agent. Consumers iterate All, Get, Keys,
// AtLeast, and Capable instead of maintaining their own agent lists.
//
// Native resume stays same-vendor. Cross-agent work is an explicit portable
// handoff. There is no transcript translation at any tier.
package agents
