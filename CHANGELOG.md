v1.55.3 (2024-05-27)
-------------------------
 * Update deps

v1.55.2 (2024-05-22)
-------------------------
 * Update elastic query DSL syntax

v1.55.1 (2024-05-20)
-------------------------
 * Use std library for errors

v1.55.0 (2024-05-20)
-------------------------
 * Move elastic utils from goflow

v1.54.9 (2024-05-09)
-------------------------
 * Allow sender id phone URNs

v1.54.8 (2024-05-09)
-------------------------
 * Ensure that new URNs are normalized and change signaure of urns.NewFromParts to take url.Values

v1.54.7 (2024-05-09)
-------------------------
 * Always trim whitespace on all parts of new URNs

v1.54.6 (2024-05-08)
-------------------------
 * Tweak urns.ParseNumber so addition of a + is a fallback

v1.54.5 (2024-05-08)
-------------------------
 * Phone URN normalization should re-parse
 * Add arg to urns.ParseNumber to determine if it allows short codes

v1.54.4 (2024-05-08)
-------------------------
 * Make phone parsing stricter

v1.54.3 (2024-05-07)
-------------------------
 * Tweak urns.NewFromParts so scheme is a string and export the urns.Schemes slice instead of exposing via function

v1.54.2 (2024-05-07)
-------------------------
 * Bring back auto adding of + to sufficiently long phone numbers when parsing URNs

v1.54.1 (2024-05-07)
-------------------------
 * Add names to schemes and make urns.Schemes() return full Scheme objects

v1.54.0 (2024-05-07)
-------------------------
 * Update deps
 * Test with both go 1.21 and 1.22
 * Refactor urns package

v1.53.2 (2024-03-28)
-------------------------
 * assertdb assert methods should return bool

v1.53.1 (2024-03-14)
-------------------------
 * Update to latest phonenumbers / protobuf

v1.53.0 (2024-03-01)
-------------------------
 * Update to chi v5

v1.52.4 (2024-02-12)
-------------------------
 * Allow mocked URL matching to be glob based

v1.52.3 (2024-01-25)
-------------------------
 * Allow any comparable type for cache.Local keys

v1.52.2 (2024-01-24)
-------------------------
 * Add a non-fetching Get, a Set and a Clear method to cache.Local

v1.52.1 (2024-01-24)
-------------------------
 * Rename cache.Cache to cache.Local for clarity

v1.52.0 (2024-01-24)
-------------------------
 * Add generic cache based on ttlcache and x/sync/singleflight
 * Add email component to webchat URNs

v1.51.2 (2024-01-15)
-------------------------
 * Panic if trying to close or start and already closed socket
 * Fix controlled closing of websockets

v1.51.1 (2024-01-12)
-------------------------
 * Allow cross site requests to websockets

v1.51.0 (2024-01-12)
-------------------------
 * Add websocket functionality to httpx

v1.50.0 (2024-01-10)
-------------------------
 * Rework support for webchat URNs, drop unused teams URNs
 * Bump golang.org/x/crypto from 0.16.0 to 0.17.0

v1.42.7 (2023-12-12)
-------------------------
 * Update deps

v1.42.6 (2023-11-24)
-------------------------
 * Update to latest phonenumbers

v1.42.5 (2023-11-20)
-------------------------
 * Update deps

v1.42.4 (2023-11-13)
-------------------------
 * Tweak stringsx.Skeleton

v1.42.3 (2023-11-08)
-------------------------
 * Update phonenumbers

v1.42.2 (2023-10-30)
-------------------------
 * Add httpx.ParseNetworks util function

v1.42.1 (2023-10-28)
-------------------------
 * Use error constants for some httpx error cases

v1.42.0 (2023-10-12)
-------------------------
 * Update to go 1.21 and update deps

v1.41.3 (2023-09-19)
-------------------------
 * Add dbutil.ScanAllJSON

v1.41.2 (2023-09-11)
-------------------------
 * Allow creating query errors without an error to wrap

v1.41.1 (2023-09-04)
-------------------------
 * Use i18n.Locale for date formatting

v1.41.0 (2023-09-04)
-------------------------
 * Move some locales code from goflow/envs

v1.40.0 (2023-08-31)
-------------------------
 * Rework syncx.Batcher so that it flushes a batch without waiting if it has enough items

v1.39.1 (2023-08-28)
-------------------------
 * Rename dbutil.Queryer to BulkQueryer for clarity

v1.39.0 (2023-08-28)
-------------------------
 * Use any instead of interface{}
 * Add dbutil.ScanAllSlice and ScanAllMap
 * Test on go 1.21

v1.38.2 (2023-08-09)
-------------------------
 * Revert validator dep upgrade

v1.38.1 (2023-08-09)
-------------------------
 * Update deps including phonenumbers

v1.38.0 (2023-08-07)
-------------------------
 * Add confusables implementation to stringsx

v1.37.0 (2023-07-20)
-------------------------
 * Storage paths shouldn't need to start with slash

v1.36.0 (2023-06-30)
-------------------------
 * Add syncx.Batcher
 * Use services for github CI

v1.35.0 (2023-02-18)
-------------------------
 * bump golang.org/x/net from 0.5.0 to 0.7.0
 * Update to latest phonenumbers
 * Remove null value support functions now that nyaruka/null has been updated

v1.34.1 (2023-01-31)
-------------------------
 * Update dependencies including phonenumbers

v1.34.0 (2023-01-26)
-------------------------
 * Add util functions for working with nullable string types

v1.33.1 (2022-11-28)
-------------------------
 * Update deps

v1.33.0 (2022-11-18)
-------------------------
 * Add util function dbutil.ToValidUTF8

v1.32.2
----------
 * Fix passing ACL to S3 puts

v1.32.1
----------
 * Update deps including phonenumbers

v1.32.0
----------
 * Storage types should have object permissions/acl set via constructor

v1.31.0
----------
 * Update httpx.DetectContentType to also return extension
 * Allow mock requestors to ignore localhost requests

v1.30.2
----------
 * MockRequestor should log requests

v1.30.1
----------
 * Time for an HTTP trace should include reading the entire body

v1.30.0
----------
 * Use go 1.19
 * Fix linter warnings
 * Add httpx.BasicAuth util

v1.29.0
----------
 * Add SantizedRequest to httpx.Trace to match SanitizedResponse

v1.28.2
----------
 * Strip more headers from reconstructed requests

v1.28.1
----------
 * Fix cloning of request bodies passed to httpx.NewRecorder

v1.28.0
----------
 * Give httpx.Recorder the option to try to reconstruct the original request

v1.27.0
----------
 * Simplify httpx.Recorder so it always dumps request first

v1.26.0
----------
 * Use pointers to httpx.MockResponse
 * Add HTTP Log support to httpx

v1.25.0
----------
 * Tweak httpx.NewMockResponse to take a byte slice

v1.24.1
----------
 * Tweak syncx naming and comments

v1.24.0
----------
 * Allow use of AWS credential chain for S3 storage

v1.23.0
----------
 * Add syncx.HashedMutexMap

v1.22.5
----------
 * Add URN type for Teams channel

v1.22.4
----------
 * Add dates.Since to match time.Since

v1.22.3
----------
 * Add mock analytics backend for testing

v1.22.2
----------
 * Update dependencies

v1.22.1
----------
 * Fix dates.Date.Combine

v1.22.0
----------
 * Add Slack Scheme

v1.21.0
----------
 * Add analytics package which provides abstraction layer for librato

v1.20.0
----------
 * Add support for db serialization to dates.Date

v1.19.1
----------
 * Update to latest phonenumbers

v1.19.0
----------
 * Update to go 1.18 and make dbutil.Bulk functions generic
 * Tidy up scheme list to make it easier to see what is there

v1.18.0
----------
 * CI with go 1.17 and 1.18
 * Add httpx.DetectContentType which wraps functionality from github.com/gabriel-vasile/mimetype

v1.17.1
----------
 * Fix race condition in S3Storage.BatchPut

v1.17.0
----------
 * Remove rcache module (replace with redisx.IntervalHash) and thus broken redigo dependency

v1.16.2
----------
 * Return QueryError if error during row iteration

v1.16.1
----------
 * Fix IsUniqueViolation for wrapped errors

v1.16.0
----------
 * Add dbutil package previously in mailroom

v1.15.1
----------
 * Add URN scheme for instagram

v1.15.0
----------
 * Make random functions threadsafe

v1.14.1
----------
 * Allow specifying max retries for S3 clients and update client library

v1.14.0
----------
 * HTTP traces should include number of retries made
 * Build and test with go 1.17

v1.13.2
----------
 * Update to latest phonenumbers

v1.13.1
----------
 * Add webchat URN scheme

v1.13.0
----------
 * Include AWS region in storage URLs

v1.12.0
----------
 * Add support for sanitizing a response trace by stripping nulls as well as invalid UTF8

v1.11.0
----------
 * Add Must* versions of jsonx.Marshal and jsonx.Unmarshal

v1.10.0
----------
 * add BatchPut to storage
 * add use of context for timeouts in storage

v1.9.2
----------
 * gsm7: Fix U+000C, form feed(\f), instead of space, for 0x0A

v1.9.1
----------
 * Use standard BCP47 (hypenated) locale codes

v1.9.0
----------
 * Add custom date formatting code from goflow and add localization support
 * Switch to go 1.16.x to get support for embed package

v1.8.0
----------
 * Allow http mocks in JSON to use actual JSON for the body

v1.7.2
----------
 * add option to save request immediately after creating recorder

v1.7.1
----------
 * ParseNumber should ignore numbers which are only possible as local numbers

v1.7.0
----------
 * Add support for IP networks in httpx.AccessConfig

v1.6.1
----------
 * Add RocketChat scheme
 * Add rcache module

v1.5.3
----------
 * Update to latest phonenumbers
 * If normalizing a number starting with a +, return it with a + if it's a possible number

v1.5.2
----------
 * Test on 1.14.x and 1.15.x

v1.5.1
----------
 * Use IsPossibleNumber instead of IsValidNumber

v1.5.0
----------
 * Add gsm7 package
 * Add httpx util for recording traces from http handlers

v1.4.0
----------
 * Add uuids package from goflow
 * Add storage package from mailroom
 * Add discord URN type

v1.3.0
----------
 * Move some util packages from goflow
 * Bump CI go versions

v1.2.0
----------
 * Add VK scheme
 * Replace Travis with github actions

v1.1.1
----------
 * Add urns.Parse function

