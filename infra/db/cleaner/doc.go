// Package cleaner deletes expired rows from tables with an expires_at column,
// such as the tables of package infra/db/migrations/keyvalue. Expired rows are
// already invisible to the drivers that own them; the cleaner reclaims space.
package cleaner
