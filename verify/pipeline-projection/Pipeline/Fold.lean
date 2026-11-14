/-
The projection in docs/superpowers/specs/2026-11-14-pipeline-projection-design.md.

`statusOf`, `failureSpec`, `lastRun`, `lastAudio`, `lastTranscript`, `lastKeyFrames`, `lastDocument`, `identityOf`, `lastCreated`, and `lastUpdated` are the spec.
Each business event is its own fact. The first accepted fact, the first start, the first ending, and the first published file are the causes.
The last fact owns the updated time.
`apply` / `fold` are the row update.
The theorems show the row update agrees with the spec.
The workflow id is derived from the job id. It is not a fact and not a field of this row.
An empty log folds to submitted so the function stays total. That result is not a job.
A saved job is a log that contains an accepted fact.
A file id is an `Ident` and a time is a `Clock`. The fold copies them and does not inspect them.
`none` is a missing file id. The service ignores the zero UUID the same way.
The service rejects an empty run id before the fact is stored.
A progress update is not a fact. It stays off this log.
A resolved, failed, canceled, or skipped step sentence updates only the clock.
A failure note that is empty or only Unicode White_Space is stored as no note, and the fold reads that as "failed".
The code points are the same set as Go's `unicode.IsSpace`.
-/

namespace Pipeline

/-- An identifier the fold copies. It is not a number. -/
structure Ident where
  label : Nat
  deriving DecidableEq, Repr, BEq

/-- A timestamp the fold copies. It is not a number. -/
structure Clock where
  label : Nat
  deriving DecidableEq, Repr, BEq

instance : OfNat Ident n where
  ofNat := { label := n }

instance : OfNat Clock n where
  ofNat := { label := n }


inductive Status where
  | submitted
  | running
  | completed
  | failed
  | canceled
  deriving DecidableEq, Repr, BEq

inductive Ending where
  | completed
  | failed (message : Option String)
  | canceled
  deriving DecidableEq, Repr, BEq

def endingStatus : Ending → Status
  | .completed => .completed
  | .failed _ => .failed
  | .canceled => .canceled

structure Identity where
  owner : Option Ident
  project : Option Ident
  inputFile : Option Ident
  idempotency : Option Ident
  deriving DecidableEq, Repr, BEq

structure Files where
  extractAudio : Option Ident
  transcribe : Option Ident
  keyFrames : Option Ident
  document : Option Ident
  deriving DecidableEq, Repr, BEq

def Files.empty : Files :=
  { extractAudio := none, transcribe := none, keyFrames := none, document := none }

inductive Fact where
  | jobAccepted (identity : Identity) (time : Clock)
  | runStarted (runId : Ident) (time : Clock)
  | runCompleted (time : Clock)
  | runFailed (message : Option String) (time : Clock)
  | runCanceled (time : Clock)
  | sourceResolved (time : Clock)
  | sourceFailed (note : Option String) (time : Clock)
  | sourceCanceled (time : Clock)
  | audioPublished (fileId : Option Ident) (time : Clock)
  | audioFailed (note : Option String) (time : Clock)
  | audioCanceled (time : Clock)
  | transcriptPublished (fileId : Option Ident) (time : Clock)
  | transcriptFailed (note : Option String) (time : Clock)
  | transcriptCanceled (time : Clock)
  | keyFramesPublished (fileId : Option Ident) (time : Clock)
  | keyFramesFailed (note : Option String) (time : Clock)
  | keyFramesCanceled (time : Clock)
  | keyFramesSkipped (time : Clock)
  | documentPublished (fileId : Option Ident) (time : Clock)
  | documentFailed (note : Option String) (time : Clock)
  | documentCanceled (time : Clock)
  deriving DecidableEq, Repr, BEq

def Fact.time : Fact → Clock
  | .jobAccepted _ time => time
  | .runStarted _ time => time
  | .runCompleted time => time
  | .runFailed _ time => time
  | .runCanceled time => time
  | .sourceResolved time => time
  | .sourceFailed _ time => time
  | .sourceCanceled time => time
  | .audioPublished _ time => time
  | .audioFailed _ time => time
  | .audioCanceled time => time
  | .transcriptPublished _ time => time
  | .transcriptFailed _ time => time
  | .transcriptCanceled time => time
  | .keyFramesPublished _ time => time
  | .keyFramesFailed _ time => time
  | .keyFramesCanceled time => time
  | .keyFramesSkipped time => time
  | .documentPublished _ time => time
  | .documentFailed _ time => time
  | .documentCanceled time => time

def Fact.ending : Fact → Option Ending
  | .runCompleted _ => some .completed
  | .runFailed message _ => some (.failed message)
  | .runCanceled _ => some .canceled
  | _ => none

def Fact.isEnding : Fact → Bool
  | .runCompleted _ | .runFailed _ _ | .runCanceled _ => true
  | _ => false

def Fact.isAccepted : Fact → Bool
  | .jobAccepted _ _ => true
  | _ => false

def Fact.isStarted : Fact → Bool
  | .runStarted _ _ => true
  | _ => false

def Status.isEnding : Status → Bool
  | .completed | .failed | .canceled => true
  | .submitted | .running => false

@[simp] theorem endingStatus_isEnding (ending : Ending) : (endingStatus ending).isEnding = true := by
  cases ending <;> rfl

/-- Unicode White_Space, the same code points as Go's `unicode.IsSpace`. -/
def Char.isNoteSpace (c : Char) : Bool :=
  let n := c.val.toNat
  n == 0x09 || n == 0x0A || n == 0x0B || n == 0x0C || n == 0x0D || n == 0x20
    || n == 0x85 || n == 0xA0 || n == 0x1680
    || (0x2000 ≤ n && n ≤ 0x200A)
    || n == 0x2028 || n == 0x2029 || n == 0x202F || n == 0x205F || n == 0x3000

def isBlank (text : String) : Bool :=
  text.all Char.isNoteSpace

/-- The note stored on a fact. Blank text is no note, same as Go's `note`. -/
def storedNote (message : Option String) : Option String :=
  match message with
  | none => none
  | some text => if text.isEmpty || isBlank text then none else some text

def failureText (message : Option String) : String :=
  match storedNote message with
  | none => "failed"
  | some text => text

def failureOf : Ending → Option String
  | .completed | .canceled => none
  | .failed message => some (failureText message)

/- The first accepted fact, the first start, the first ending, and the first
file published for a step are the causes. A later copy does not replace them.
A start owns status only while no ending has been recorded. -/

def lastEnding : List Fact → Option Ending
  | [] => none
  | fact :: facts =>
    match fact.ending with
    | some ending => some ending
    | none => lastEnding facts

def hasStarted : List Fact → Bool
  | [] => false
  | .runStarted _ _ :: _ => true
  | _ :: facts => hasStarted facts

def statusOf (facts : List Fact) : Status :=
  match lastEnding facts with
  | some ending => endingStatus ending
  | none => if hasStarted facts then .running else .submitted

def lastRun : List Fact → Option Ident
  | [] => none
  | .runStarted runId _ :: _ => some runId
  | _ :: facts => lastRun facts

def failureSpec (facts : List Fact) : Option String :=
  match lastEnding facts with
  | some ending => failureOf ending
  | none => none

def lastSubmitted : List Fact → Option Identity
  | [] => none
  | .jobAccepted id _ :: _ => some id
  | _ :: facts => lastSubmitted facts

def lastCreated : List Fact → Option Clock
  | [] => none
  | .jobAccepted _ time :: _ => some time
  | _ :: facts => lastCreated facts

/-- The last fact the fold reads owns the clock, even when its time is earlier. -/
def lastUpdated : List Fact → Option Clock
  | [] => none
  | fact :: facts =>
    match lastUpdated facts with
    | some time => some time
    | none => some fact.time

def lastAudio : List Fact → Option Ident
  | [] => none
  | .audioPublished (some id) _ :: _ => some id
  | _ :: facts => lastAudio facts

def lastTranscript : List Fact → Option Ident
  | [] => none
  | .transcriptPublished (some id) _ :: _ => some id
  | _ :: facts => lastTranscript facts

def lastKeyFrames : List Fact → Option Ident
  | [] => none
  | .keyFramesPublished (some id) _ :: _ => some id
  | _ :: facts => lastKeyFrames facts

def lastDocument : List Fact → Option Ident
  | [] => none
  | .documentPublished (some id) _ :: _ => some id
  | _ :: facts => lastDocument facts

theorem lastEnding_snoc (facts : List Fact) (fact : Fact) :
    lastEnding (facts ++ [fact]) =
      match lastEnding facts with
      | some ending => some ending
      | none => fact.ending := by
  induction facts with
  | nil => cases fact <;> rfl
  | cons head _ ih =>
    simp only [List.cons_append, lastEnding]
    cases h : head.ending with
    | some _ => all_goals first | simp | simp [ih] | simp [h]
    | none => simp [ih, Fact.ending]

theorem hasStarted_snoc (facts : List Fact) (fact : Fact) :
    hasStarted (facts ++ [fact]) = (hasStarted facts || fact.isStarted) := by
  induction facts with
  | nil => cases fact <;> rfl
  | cons head _ ih =>
    cases head <;> simp [hasStarted, ih, List.cons_append]

theorem lastRun_snoc (facts : List Fact) (fact : Fact) :
    lastRun (facts ++ [fact]) =
      match lastRun facts with
      | some runId => some runId
      | none =>
        match fact with
        | .runStarted runId _ => some runId
        | _ => none := by
  induction facts with
  | nil => cases fact <;> rfl
  | cons head _ ih =>
    cases head <;> simp [lastRun, ih, List.cons_append]

/- Row update. One fact, then the fold from an empty row. -/

def stepStatus (status : Status) : Fact → Status
  | .runCompleted _ => if status.isEnding then status else .completed
  | .runFailed _ _ => if status.isEnding then status else .failed
  | .runCanceled _ => if status.isEnding then status else .canceled
  | .runStarted _ _ => if status.isEnding then status else .running
  | .jobAccepted _ _ => if status.isEnding || status = .running then status else .submitted
  | _ => status

def stepRun (runId : Option Ident) : Fact → Option Ident
  | .runStarted id _ => some (runId.getD id)
  | _ => runId

/-- The first published id stays. A missing id does not clear it. -/
def keepFile (current incoming : Option Ident) : Option Ident :=
  current.orElse (fun () => incoming)

def stepFile (files : Files) : Fact → Files
  | .audioPublished id _ => { files with extractAudio := keepFile files.extractAudio id }
  | .transcriptPublished id _ => { files with transcribe := keepFile files.transcribe id }
  | .keyFramesPublished id _ => { files with keyFrames := keepFile files.keyFrames id }
  | .documentPublished id _ => { files with document := keepFile files.document id }
  | _ => files

def stepFailure (failure : Option String) (status : Status) : Fact → Option String
  | fact =>
    match fact.ending with
    | some ending => if status.isEnding then failure else failureOf ending
    | none => failure

def stepIdentity (identity : Identity) (created : Option Clock) : Fact → Identity
  | .jobAccepted id _ =>
    match created with
    | none => id
    | some _ => identity
  | _ => identity

def stepCreated (created : Option Clock) : Fact → Option Clock
  | .jobAccepted _ time =>
    match created with
    | none => some time
    | some kept => some kept
  | _ => created

structure Row where
  status : Status
  failure : Option String
  runId : Option Ident
  files : Files
  identity : Identity
  dateCreated : Option Clock
  dateUpdated : Option Clock
  deriving DecidableEq, Repr, BEq

def Identity.empty : Identity :=
  { owner := none, project := none, inputFile := none, idempotency := none }

def identityOf (facts : List Fact) : Identity :=
  match lastSubmitted facts with
  | some id => id
  | none => Identity.empty

def Row.empty : Row where
  status := .submitted
  failure := none
  runId := none
  files := Files.empty
  identity := Identity.empty
  dateCreated := none
  dateUpdated := none

def apply (row : Row) (fact : Fact) : Row where
  status := stepStatus row.status fact
  failure := stepFailure row.failure row.status fact
  runId := stepRun row.runId fact
  files := stepFile row.files fact
  identity := stepIdentity row.identity row.dateCreated fact
  dateCreated := stepCreated row.dateCreated fact
  dateUpdated := some fact.time

def fold (facts : List Fact) : Row :=
  facts.foldl apply Row.empty

def hasEnding (facts : List Fact) : Bool :=
  facts.any Fact.isEnding

/-- A second ending is not stored. The first one stays. -/
def append (log : List Fact) (fact : Fact) : List Fact :=
  if fact.isEnding && hasEnding log then log else log ++ [fact]

/-- The fact is stored only when the row update commits. -/
def commit (log : List Fact) (fact : Fact) (rowUpdated : Bool) : List Fact :=
  if rowUpdated then append log fact else log

theorem statusOf_nil : statusOf [] = .submitted := by
  simp [statusOf, lastEnding, hasStarted]

/-- A saved job is a log that contains an accepted fact. The empty fold is not one. -/
def IsJob (facts : List Fact) : Prop :=
  facts.any Fact.isAccepted = true

theorem empty_log_is_not_a_job : ¬ IsJob [] := by
  simp [IsJob]

theorem statusOf_append (facts : List Fact) (fact : Fact) :
    statusOf (facts ++ [fact]) = stepStatus (statusOf facts) fact := by
  cases fact <;> cases h : lastEnding facts <;> try cases ‹Ending›
  all_goals
    cases hs : hasStarted facts <;>
      simp [statusOf, stepStatus, lastEnding_snoc, hasStarted_snoc, h, hs, Fact.isStarted, Fact.ending, Status.isEnding, endingStatus]

theorem foldl_apply_status (facts : List Fact) (row : Row) :
    (facts.foldl apply row).status = facts.foldl stepStatus row.status := by
  induction facts generalizing row with
  | nil => rfl
  | cons _ _ ih =>
    unfold List.foldl
    rw [ih]
    rfl

theorem foldl_from_statusOf (earlier suffix : List Fact) :
    suffix.foldl stepStatus (statusOf earlier) = statusOf (earlier ++ suffix) := by
  induction suffix generalizing earlier with
  | nil => simp
  | cons fact suffix ih =>
    unfold List.foldl
    rw [← statusOf_append earlier fact]
    rw [ih (earlier ++ [fact])]
    simp [List.append_assoc]

theorem fold_status_eq (facts : List Fact) : (fold facts).status = statusOf facts := by
  have hspec := foldl_from_statusOf [] facts
  have happly := foldl_apply_status facts Row.empty
  simp only [statusOf_nil, List.nil_append, fold, Row.empty] at hspec happly ⊢
  exact happly.trans hspec

theorem foldl_apply_files (facts : List Fact) (row : Row) :
    (facts.foldl apply row).files = facts.foldl stepFile row.files := by
  induction facts generalizing row with
  | nil => rfl
  | cons _ _ ih =>
    unfold List.foldl
    rw [ih]
    rfl

theorem foldl_audio (facts : List Fact) (files : Files) :
    (facts.foldl stepFile files).extractAudio =
      match files.extractAudio with
      | some id => some id
      | none => lastAudio facts := by
  induction facts generalizing files with
  | nil =>
    simp [List.foldl]
    cases files.extractAudio <;> rfl
  | cons fact _ ih =>
    unfold List.foldl
    rw [ih]
    cases fact with
    | audioPublished id _ =>
      cases id <;> cases h : files.extractAudio <;> simp [stepFile, keepFile, lastAudio, h]
    | _ => cases h : files.extractAudio <;> simp [stepFile, keepFile, lastAudio, h]

theorem fold_audio (facts : List Fact) :
    (fold facts).files.extractAudio = lastAudio facts := by
  have hfiles := foldl_apply_files facts Row.empty
  have hstep := foldl_audio facts Files.empty
  unfold fold
  simp only [Row.empty] at hfiles ⊢
  rw [hfiles]
  simpa [Files.empty] using hstep

theorem foldl_transcript (facts : List Fact) (files : Files) :
    (facts.foldl stepFile files).transcribe =
      match files.transcribe with
      | some id => some id
      | none => lastTranscript facts := by
  induction facts generalizing files with
  | nil =>
    simp [List.foldl]
    cases files.transcribe <;> rfl
  | cons fact _ ih =>
    unfold List.foldl
    rw [ih]
    cases fact with
    | transcriptPublished id _ =>
      cases id <;> cases h : files.transcribe <;> simp [stepFile, keepFile, lastTranscript, h]
    | _ => cases h : files.transcribe <;> simp [stepFile, keepFile, lastTranscript, h]

theorem fold_transcript (facts : List Fact) :
    (fold facts).files.transcribe = lastTranscript facts := by
  have hfiles := foldl_apply_files facts Row.empty
  have hstep := foldl_transcript facts Files.empty
  unfold fold
  simp only [Row.empty] at hfiles ⊢
  rw [hfiles]
  simpa [Files.empty] using hstep

theorem foldl_keyFrames (facts : List Fact) (files : Files) :
    (facts.foldl stepFile files).keyFrames =
      match files.keyFrames with
      | some id => some id
      | none => lastKeyFrames facts := by
  induction facts generalizing files with
  | nil =>
    simp [List.foldl]
    cases files.keyFrames <;> rfl
  | cons fact _ ih =>
    unfold List.foldl
    rw [ih]
    cases fact with
    | keyFramesPublished id _ =>
      cases id <;> cases h : files.keyFrames <;> simp [stepFile, keepFile, lastKeyFrames, h]
    | _ => cases h : files.keyFrames <;> simp [stepFile, keepFile, lastKeyFrames, h]

theorem fold_keyFrames (facts : List Fact) :
    (fold facts).files.keyFrames = lastKeyFrames facts := by
  have hfiles := foldl_apply_files facts Row.empty
  have hstep := foldl_keyFrames facts Files.empty
  unfold fold
  simp only [Row.empty] at hfiles ⊢
  rw [hfiles]
  simpa [Files.empty] using hstep

theorem foldl_document (facts : List Fact) (files : Files) :
    (facts.foldl stepFile files).document =
      match files.document with
      | some id => some id
      | none => lastDocument facts := by
  induction facts generalizing files with
  | nil =>
    simp [List.foldl]
    cases files.document <;> rfl
  | cons fact _ ih =>
    unfold List.foldl
    rw [ih]
    cases fact with
    | documentPublished id _ =>
      cases id <;> cases h : files.document <;> simp [stepFile, keepFile, lastDocument, h]
    | _ => cases h : files.document <;> simp [stepFile, keepFile, lastDocument, h]

theorem fold_document (facts : List Fact) :
    (fold facts).files.document = lastDocument facts := by
  have hfiles := foldl_apply_files facts Row.empty
  have hstep := foldl_document facts Files.empty
  unfold fold
  simp only [Row.empty] at hfiles ⊢
  rw [hfiles]
  simpa [Files.empty] using hstep

theorem foldl_apply_run (facts : List Fact) (row : Row) :
    (facts.foldl apply row).runId = facts.foldl stepRun row.runId := by
  induction facts generalizing row with
  | nil => rfl
  | cons _ _ ih =>
    unfold List.foldl
    rw [ih]
    rfl

theorem foldl_stepRun_spec (facts : List Fact) (current : Option Ident) :
    facts.foldl stepRun current =
      match current with
      | some _ => current
      | none => lastRun facts := by
  induction facts generalizing current with
  | nil => cases current <;> rfl
  | cons fact facts ih =>
    unfold List.foldl
    cases current with
    | some _ => cases fact <;> simp [stepRun, ih]
    | none =>
      rw [ih]
      cases fact <;> simp [stepRun, lastRun]

theorem fold_run_eq (facts : List Fact) : (fold facts).runId = lastRun facts := by
  have happly := foldl_apply_run facts Row.empty
  have hspec := foldl_stepRun_spec facts none
  simp only [fold, Row.empty] at happly ⊢
  rw [happly, hspec]

theorem fold_append_singleton (facts : List Fact) (fact : Fact) :
    fold (facts ++ [fact]) = apply (fold facts) fact := by
  simp [fold, List.foldl_append]

theorem failureSpec_snoc (facts : List Fact) (fact : Fact) :
    failureSpec (facts ++ [fact]) = stepFailure (failureSpec facts) (statusOf facts) fact := by
  cases fact <;> cases h : lastEnding facts <;> try cases ‹Ending›
  all_goals
    cases hs : hasStarted facts <;>
      simp [failureSpec, stepFailure, statusOf, lastEnding_snoc, h, hs, Fact.ending, Status.isEnding, endingStatus, failureOf]

theorem fold_failure_eq (facts : List Fact) : (fold facts).failure = failureSpec facts := by
  induction hrev : facts.reverse generalizing facts with
  | nil =>
    simp [List.reverse_eq_nil_iff] at hrev
    subst hrev
    rfl
  | cons fact rev ih =>
    have hfacts : facts = rev.reverse ++ [fact] := by
      simpa [List.reverse_reverse, List.reverse_cons] using congrArg List.reverse hrev
    have ih' := ih rev.reverse (by simp)
    rw [hfacts, fold_append_singleton]
    simp [apply]
    rw [ih', fold_status_eq]
    rw [← failureSpec_snoc]

theorem lastSubmitted_snoc (facts : List Fact) (fact : Fact) :
    lastSubmitted (facts ++ [fact]) =
      match lastSubmitted facts with
      | some id => some id
      | none =>
        match fact with
        | .jobAccepted id _ => some id
        | _ => none := by
  induction facts with
  | nil => cases fact <;> rfl
  | cons head _ ih =>
    cases head <;> simp [lastSubmitted, ih, List.cons_append]

theorem first_submitted_created (facts : List Fact) :
    (lastSubmitted facts).isSome = (lastCreated facts).isSome := by
  induction facts with
  | nil => rfl
  | cons fact _ ih =>
    cases fact <;> simp [lastSubmitted, lastCreated, ih]

theorem identityOf_snoc (facts : List Fact) (fact : Fact) :
    identityOf (facts ++ [fact]) =
      stepIdentity (identityOf facts) (lastCreated facts) fact := by
  cases fact with
  | jobAccepted id _ =>
    cases h : lastSubmitted facts with
    | some _ =>
      have hcreated : (lastCreated facts).isSome = true := by
        simpa [h] using first_submitted_created facts
      cases hc : lastCreated facts with
      | none => simp [hc, Option.isSome] at hcreated
      | some _ =>
        simp [identityOf, stepIdentity, lastSubmitted_snoc, h]
    | none =>
      have hcreated : lastCreated facts = none := by
        have hsome := first_submitted_created facts
        cases hc : lastCreated facts <;> simp [h, hc, Option.isSome] at hsome
        rfl
      simp [identityOf, stepIdentity, lastSubmitted_snoc, h, hcreated]
  | _ =>
    cases h : lastSubmitted facts <;>
      simp [identityOf, stepIdentity, lastSubmitted_snoc, h]

theorem lastCreated_snoc (facts : List Fact) (fact : Fact) :
    lastCreated (facts ++ [fact]) =
      match lastCreated facts with
      | some time => some time
      | none =>
        match fact with
        | .jobAccepted _ time => some time
        | _ => none := by
  induction facts with
  | nil => cases fact <;> rfl
  | cons head _ ih =>
    cases head <;> simp [lastCreated, ih, List.cons_append]

theorem fold_created_eq (facts : List Fact) : (fold facts).dateCreated = lastCreated facts := by
  induction hrev : facts.reverse generalizing facts with
  | nil =>
    simp [List.reverse_eq_nil_iff] at hrev
    subst hrev
    rfl
  | cons fact rev ih =>
    have hfacts : facts = rev.reverse ++ [fact] := by
      simpa [List.reverse_reverse, List.reverse_cons] using congrArg List.reverse hrev
    have ih' := ih rev.reverse (by simp)
    rw [hfacts, fold_append_singleton]
    simp [apply]
    rw [ih']
    cases fact <;> cases hcreated : lastCreated rev.reverse <;> simp [stepCreated, lastCreated_snoc, hcreated]

theorem fold_identity_eq (facts : List Fact) : (fold facts).identity = identityOf facts := by
  induction hrev : facts.reverse generalizing facts with
  | nil =>
    simp [List.reverse_eq_nil_iff] at hrev
    subst hrev
    rfl
  | cons fact rev ih =>
    have hfacts : facts = rev.reverse ++ [fact] := by
      simpa [List.reverse_reverse, List.reverse_cons] using congrArg List.reverse hrev
    have ih' := ih rev.reverse (by simp)
    rw [hfacts, fold_append_singleton]
    simp [apply]
    rw [ih', fold_created_eq, identityOf_snoc]

theorem lastUpdated_snoc (facts : List Fact) (fact : Fact) :
    lastUpdated (facts ++ [fact]) = some fact.time := by
  induction facts with
  | nil => cases fact <;> rfl
  | cons _ tail ih =>
    simp only [List.cons_append]
    conv =>
      lhs
      unfold lastUpdated
    rw [ih]

theorem fold_updated_eq (facts : List Fact) : (fold facts).dateUpdated = lastUpdated facts := by
  induction hrev : facts.reverse generalizing facts with
  | nil =>
    simp [List.reverse_eq_nil_iff] at hrev
    subst hrev
    rfl
  | cons fact rev _ =>
    have hfacts : facts = rev.reverse ++ [fact] := by
      simpa [List.reverse_reverse, List.reverse_cons] using congrArg List.reverse hrev
    rw [hfacts, fold_append_singleton, lastUpdated_snoc]
    simp [apply]

/- Corollaries named by the spec. -/

theorem started_after_ending_keeps_status (facts : List Fact) (runId : Ident) (time : Clock)
    (h : (fold facts).status.isEnding = true) :
    (fold (facts ++ [.runStarted runId time])).status = (fold facts).status := by
  rw [fold_status_eq] at h
  rw [fold_status_eq, fold_status_eq, statusOf_append]
  simp [stepStatus, h]

theorem append_second_ending (log : List Fact) (fact : Fact)
    (hFact : fact.isEnding = true) (hLog : hasEnding log = true) :
    append log fact = log := by
  simp [append, hFact, hLog]

theorem commit_rollback (log : List Fact) (fact : Fact) : commit log fact false = log := by
  simp [commit]

theorem commit_ok_appends (log : List Fact) (fact : Fact) (h : hasEnding log = false) :
    commit log fact true = log ++ [fact] := by
  simp [commit, append, h]

/- A stable id is written once. A repeated submit, start, or ending keeps the first payload. -/

inductive Slot where
  | submit
  | start
  | run
  | fresh (n : Nat)
  deriving DecidableEq, Repr, BEq

structure Written where
  slot : Slot
  fact : Fact
  deriving DecidableEq, Repr, BEq

def factsOf (log : List Written) : List Fact :=
  log.map fun written => written.fact

def slotTaken (log : List Written) (slot : Slot) : Bool :=
  log.any fun written => written.slot == slot

/-- The writer. A second ending is dropped. A repeated slot keeps the stored fact. -/
def record (log : List Written) (slot : Slot) (fact : Fact) : List Written :=
  if fact.isEnding && hasEnding (factsOf log) then log
  else if slotTaken log slot then log
  else log ++ [{ slot := slot, fact := fact }]

theorem record_taken (log : List Written) (slot : Slot) (fact : Fact)
    (h : slotTaken log slot = true) : record log slot fact = log := by
  unfold record
  cases fact.isEnding && hasEnding (factsOf log) <;> simp [h]

theorem record_second_ending (log : List Written) (slot : Slot) (fact : Fact)
    (hFact : fact.isEnding = true) (hLog : hasEnding (factsOf log) = true) :
    record log slot fact = log := by
  simp [record, hFact, hLog]

/- Checked examples from the spec. -/

def sampleId : Identity :=
  { owner := some 1, project := some 2, inputFile := some 3, idempotency := some 4 }

#guard statusOf [] == .submitted
#guard (fold [.jobAccepted sampleId 0, .runStarted 8 1]).status == .running
#guard (fold [.jobAccepted sampleId 0, .runStarted 8 1]).dateUpdated == some 1
#guard (fold [.runStarted 8 0, .jobAccepted sampleId 1]).status == .running
#guard (fold [.runStarted 8 0, .jobAccepted sampleId 1]).dateUpdated == some 1
#guard (fold [.runStarted 8 0, .jobAccepted sampleId 1]).runId == some 8
#guard (fold [.runStarted 1 0, .documentPublished (some 5) 1]).status == .running
#guard (fold [.runStarted 1 0, .sourceResolved 1]).status == .running
#guard (fold [.runStarted 1 0, .sourceResolved 1]).files.extractAudio == none
#guard (fold [.sourceResolved 0, .audioPublished (some 7) 1]).files.extractAudio == some 7
#guard (fold [.runCompleted 0, .runStarted 9 1]).status == .completed
#guard (fold [.runCompleted 0, .runStarted 9 1]).runId == some 9
#guard (fold [.runCompleted 0, .audioFailed none 1]).status == .completed
#guard (fold [.runFailed none 0]).failure == some "failed"
#guard (fold [.runFailed (some "") 0]).failure == some "failed"
#guard (fold [.runFailed (some "  ") 0]).failure == some "failed"
#guard (fold [.runFailed (some "\u00a0") 0]).failure == some "failed"
#guard (fold [.runFailed (some "\u2003\u000b") 0]).failure == some "failed"
#guard (fold [.runFailed (some "  boom") 0]).failure == some "  boom"
#guard (fold [.runFailed (some "\u00a0boom") 0]).failure == some "\u00a0boom"
#guard (fold [.runFailed (some "boom") 0, .runCanceled 1]).failure == some "boom"
#guard (fold [.runFailed (some "boom") 0, .runCanceled 1]).status == .failed
#guard (fold [.audioPublished (some 7) 0, .audioPublished (some 8) 1]).files.extractAudio == some 7
#guard (fold [.audioPublished none 0, .audioPublished (some 8) 1]).files.extractAudio == some 8
#guard (fold [.audioPublished none 0]).files.extractAudio == none
#guard ([].any Fact.isAccepted) == false
#guard (fold [.audioPublished (some 7) 0, .sourceResolved 1]).files.extractAudio == some 7
#guard (fold [.audioPublished (some 7) 0, .sourceResolved 1]).dateUpdated == some 1
#guard (fold [.runCompleted 0, .sourceResolved 1]).status == .completed
#guard (fold [.runCompleted 0, .sourceResolved 1]).failure == none
#guard (fold [.sourceResolved 1]).dateUpdated == some 1
#guard (fold []).files.document == none
#guard (([] : List Fact).foldl apply { Row.empty with files.document := some 99 }).files.document == some 99
#guard append [.runCompleted 0] (.runFailed (some "later") 1) == [.runCompleted 0]
#guard commit [.jobAccepted sampleId 0] (.runCompleted 1) false == [.jobAccepted sampleId 0]
#guard (fold [.jobAccepted sampleId 4]).identity == sampleId
#guard (fold [.jobAccepted sampleId 4]).dateCreated == some 4
#guard (fold [.runStarted 6 2]).runId == some 6
#guard (fold [.jobAccepted sampleId 4, .jobAccepted { sampleId with owner := some 9 } 5]).identity.owner == some 1
#guard (fold [.runFailed (some "boom") 1]).failure == failureSpec [.runFailed (some "boom") 1]

def startedOnce : List Written := record [] .start (.runStarted 1 0)
def submittedOnce : List Written := record [] .submit (.jobAccepted sampleId 1)
def endedOnce : List Written := record [] .run (.runCompleted 0)

#guard record startedOnce .start (.runStarted 9 4) == startedOnce
#guard (fold (factsOf (record startedOnce .start (.runStarted 9 4)))).runId == some 1
#guard record submittedOnce .submit (.jobAccepted { sampleId with owner := some 9 } 8) == submittedOnce
#guard (fold (factsOf (record submittedOnce .submit (.jobAccepted { sampleId with owner := some 9 } 8)))).identity == sampleId
#guard record endedOnce (.fresh 0) (.runFailed (some "later") 3) == endedOnce

end Pipeline
