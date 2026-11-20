// Package keyvalue creates the UNLOGGED tables of the Postgres drivers of
// cache, csrf and lock, which replace Redis or Valkey. Run it only when the
// application uses those drivers.
//
// UNLOGGED tables are fast, but Postgres empties them after a crash (every
// session is gone) and they cannot be read on a standby.
package keyvalue
