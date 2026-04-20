-- One-shot dev-environment cleanup for hackathons that violate the post-#27/#28
-- validation rules. NOT a migration — run manually after backing up data/forgejo.db.
-- See docs/superpowers/specs/2026-04-20-judge-score-feedback-design.md § E.

BEGIN;

-- Published hackathons lacking criteria at time of script authoring (2026-04-20).
-- If re-running later, edit this list or extend with a query.
CREATE TEMP TABLE bad_hackathons AS
  SELECT id FROM hackathon WHERE id IN (87, 72, 71, 76, 79);

CREATE TEMP TABLE bad_tracks AS
  SELECT id FROM hackathon_track WHERE hackathon_id IN (SELECT id FROM bad_hackathons);

-- Delete leaf rows first (SQLite foreign_keys PRAGMA is off by default in Forgejo).
DELETE FROM hackathon_judge_score       WHERE hackathon_id IN (SELECT id FROM bad_hackathons);
DELETE FROM hackathon_track_criteria    WHERE track_id     IN (SELECT id FROM bad_tracks);
DELETE FROM hackathon_judge_criteria    WHERE hackathon_id IN (SELECT id FROM bad_hackathons);
DELETE FROM hackathon_submission        WHERE hackathon_id IN (SELECT id FROM bad_hackathons);
DELETE FROM hackathon_registration      WHERE hackathon_id IN (SELECT id FROM bad_hackathons);
DELETE FROM hackathon_judge             WHERE hackathon_id IN (SELECT id FROM bad_hackathons);
DELETE FROM hackathon_track             WHERE hackathon_id IN (SELECT id FROM bad_hackathons);
DELETE FROM phase                       WHERE activity_kind = 'hackathon'
                                          AND activity_id IN (SELECT id FROM bad_hackathons);
DELETE FROM hackforger_action           WHERE entity_type = 'hackathon'
                                          AND entity_id   IN (SELECT id FROM bad_hackathons);
DELETE FROM hackathon                   WHERE id           IN (SELECT id FROM bad_hackathons);

-- Orphan scored feed events from any hackathon (op_type=33 with zero matching scores).
DELETE FROM hackforger_action
 WHERE op_type = 33
   AND NOT EXISTS (
     SELECT 1 FROM hackathon_judge_score
      WHERE hackathon_id = hackforger_action.entity_id
        AND judge_id     = hackforger_action.user_id
   );

COMMIT;
