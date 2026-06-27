// Package client provides a KiwiVM API client for BandwagonHost VPS instances.
//
// Use [NewClient] with a KiwiVM API key and VEID, then pass a
// [context.Context] to each method. Read-only methods use GET requests.
// Methods that change VPS state use POST requests with
// application/x-www-form-urlencoded form data. For those write calls, request
// parameters and credentials are not placed in the URL.
//
// KiwiVM returns an error field in every API response. The client converts
// non-zero API errors into [BWHError], including optional locking details when
// the VPS is busy. Use [GetBWHError] or errors.As to inspect structured API
// errors.
//
// Some methods can affect service availability, networking, credentials, abuse
// state, or stored data. Callers should add their own confirmation and
// validation around write methods such as [Client.Kill], [Client.ReinstallOS],
// [Client.ResetRootPassword], [Client.RestoreSnapshot], and
// [Client.StartMigration].
package client
