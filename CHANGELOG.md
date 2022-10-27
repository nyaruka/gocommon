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

