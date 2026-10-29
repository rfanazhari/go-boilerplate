# Decision Log: Blogs

Business-driven technical decisions for the planned Blogs feature. Business authority is the [approved capability](../../business/capabilities/blogs.md#approved-decisions--2026-10-29), dated 2026-10-29. Business outcomes below were approved there; implementation details were selected in this tripack and are not claims of separate stakeholder approval or completed implementation. Engineering details live in [technical design](technical-design.md).

## D1: Keep deleted writing for administrator review

- **Business driver:** BR-TASKDEMO-066, 073–074 and OQ-09 require withdrawal from ordinary readers while preserving administrator access.
- **Technical decision:** Set a deletion timestamp on the existing blog record. Keep publication status separate and retain the record and its slug. Only the administrator list/detail can reveal deleted writing. No restore or permanent deletion is included.
- **What this means for the product:** Deleted posts disappear from public readers and the author, while administrators can still inspect them. An author cannot republish a deleted post. Administrators repeating deletion receive success; authors repeating deletion receive not-found.
- **Alternatives considered:** Physical deletion loses the required review record. A third publication status mixes withdrawal state with publication and complicates the approved draft/published model.
- **Trade-offs accepted:** Deleted content continues to consume storage. There is no automatic retention deadline or restoration flow. Destructive database rollback would lose these records and slug reservations, so application rollback normally retains the table.
- **Revisit when:** The business defines retention, legal erasure, restoration, or permanent removal requirements.
- **Status:** active.

## D2: Give public readers and authenticated users separate access paths

- **Business driver:** BR-TASKDEMO-053–054 and 070–074 require anonymous published reads, private drafts, and administrator-only deleted reads. Being logged in must not reveal hidden content on a public page.
- **Technical decision:** Public GET `/blogs` and `/blogs/{slug}` always apply public visibility without checking reader credentials. Management/private access uses `/private/blogs/{id}` and action paths; administrator reads use `/admin/blogs`. Creation is authenticated POST `/blogs`. Existing login remains the only credential mechanism.
- **What this means for the product:** The public URL shows the same permitted content to everyone, including authors and administrators. Authors use their private detail route to read drafts; administrators use the administrator surface to inspect deleted posts. Inactive/deleted accounts cannot use protected access but may read public posts anonymously.
- **Alternatives considered:** Giving one GET route optional authentication risks making its visibility change with reader identity. Requiring login for every blog read would contradict public readership.
- **Trade-offs accepted:** Clients integrate distinct access paths. Tests must verify route registration and prevent the protected route group from intercepting public GET requests.
- **Revisit when:** Product requests private previews, shared draft links, or a different administrator experience.
- **Status:** active.

## D3: Use a readable, permanent public address and UUIDs for management

- **Business driver:** BR-TASKDEMO-084 and OQ-06–07 require a title-based public address, duplicate titles, permanent URLs after renaming, and no address reuse after deletion.
- **Technical decision:** Create a lowercase title base plus a 12-character server-generated UUID suffix once. Use an 80-character base limit and `blog` fallback where normalization yields no Latin letters/digits. Public detail accepts only this slug; private/admin/actions use the full UUID. Keep global slug uniqueness, including deleted rows, and retry generated collisions up to five attempts.
- **What this means for the product:** Renaming does not break shared links. Two authors can use the same title. A deleted link cannot later point to a different post. Non-Latin-only titles are accepted but may receive a generic `blog-...` address.
- **Alternatives considered:** UUID-only public URLs fail the title-based requirement. Regenerating addresses on rename breaks existing links. Using the title alone forces unnecessary title uniqueness or numbering conventions.
- **Trade-offs accepted:** Addresses retain their original wording after edits. The short suffix reduces readability slightly, and rare collisions require retry handling. The suffix is not a privacy mechanism.
- **Revisit when:** The business requires localized public addresses, custom aliases, or explicit redirects after an address change.
- **Status:** active.

## D4: Start with numbered pages and small summaries

- **Business driver:** OQ-13 approves page/pageSize, 20 items by default, a maximum of 100, newest-created first with an ID tie-breaker, and full content only on detail. OQ-09 also requires an administrator list.
- **Technical decision:** Use page 1 by default, pageSize 20, a strict 1–100 size limit, and descending creation time then descending ID. Return summaries and total count. Apply filtering before paging. Administrators use a separate deletion filter: active, deleted, or all; the default excludes deleted records.
- **What this means for the product:** Readers can navigate numbered pages with compact results. Administrators can intentionally include removed posts. Items with identical creation times have consistent ordering within an unchanged dataset.
- **Alternatives considered:** Returning every post becomes expensive and conflicts with the approved pagination. Cursor navigation can handle large moving datasets better but does not provide the approved numbered-page interface directly.
- **Trade-offs accepted:** New publications or withdrawals can shift items between requests, and large offsets/counts may become slower. Count and items within one database response share a snapshot; separate page requests do not.
- **Revisit when:** Measured volume or navigation behavior makes numbered pages/counting too costly, or readers need a continuous feed.
- **Status:** active.

## D5: Derive public availability from the author's current account

- **Business driver:** BR-TASKDEMO-075 and OQ-08 require posts to disappear when the author is inactive/deleted, without rewriting publication status, and reappear after eligible reactivation.
- **Technical decision:** Public list/detail read the blog and current account together. Require published, blog not deleted, author active, and author not deleted on every public read. Do not add account-to-blog status updates or cache public eligibility in the initial feature.
- **What this means for the product:** Revoking an author's account removes their posts from subsequent public reads. Eligible reactivation restores still-published, non-deleted posts without manual republishing. Existing account operations keep their contract.
- **Alternatives considered:** Automatically unpublishing every post loses the approved publication state. Checking account state only at publish time leaves old posts visible after account withdrawal. Background synchronization introduces a visibility delay.
- **Trade-offs accepted:** Public reads consult account state, increasing query work. A response already being read while an account changes uses its database snapshot; a new read after the change commits reflects it.
- **Revisit when:** Product requests permanent withdrawal after deactivation, introduces caching, or requires stronger guarantees for requests already in flight.
- **Status:** active.

## D6: Count readable text and sanitize every stored version

- **Business driver:** BR-TASKDEMO-080–083 and OQ-03–05/12 allow formatted writing, short/empty drafts, published text of 100–1000 characters, and immediate valid edits to published posts.
- **Technical decision:** Enforce the closed p/br/h1–h5/a policy on the server, keep only safe absolute HTTP/HTTPS links, and store/return sanitized HTML. Count normalized readable Unicode text after sanitization rather than markup length. Drafts may be empty but cannot exceed 1000; publication requires at least 100. Validate a complete candidate edit before changing any stored fields.
- **What this means for the product:** Authors can use the approved formatting without padding length with markup. Unsupported formatting is removed. Valid edits go live immediately; invalid edits leave the prior published version intact.
- **Alternatives considered:** Plain text removes required formatting. Client-only filtering fails to enforce one consistent server policy. Counting raw HTML makes author-visible limits depend on invisible markup.
- **Trade-offs accepted:** Sanitization may remove pasted formatting/links. The server needs careful HTML security and boundary tests. There is no separate approval or version history for direct published edits.
- **Revisit when:** Product adds images, richer formatting, editorial review, version history, or a different content-length policy.
- **Status:** active.

## D7: Keep authorship separate from moderation

- **Business driver:** OQ-01 explicitly prohibits administrator creation. BR-TASKDEMO-052/062–067 and OQ-10 give authors control over wording/publication while moderators can withdraw content.
- **Technical decision:** Role user alone can create. Ownership controls editing/publishing; owner or administrator can unpublish/delete. Verify permission before treating repeated operations as successful. Administrator unpublish returns a live post to draft and does not create an approval requirement for republishing.
- **What this means for the product:** Administrators cannot write new posts or rewrite/publish another author's work. An author may republish after moderator unpublish. Soft deletion is the approved way to prevent the author from republishing that post.
- **Alternatives considered:** A general administrator override would violate authorship rules. A moderation approval queue would add an unapproved workflow. Reusing task permissions would not express administrator withdrawal or public/private blog visibility.
- **Trade-offs accepted:** Unpublish alone is temporary moderation because the author can republish. No moderation reason or notification is recorded in the initial feature.
- **Revisit when:** The business introduces editorial accounts, permanent publication bans without deletion, approval workflows, or moderation audit requirements.
- **Status:** active.
