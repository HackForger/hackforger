# Judge Stories

### Story H-008: Score Submissions (Multi-Criteria)

- **Summary:** Judge scores each submission against weighted criteria

#### Use Case:
- **As a** hackathon judge
- **I want to** score each submission on the defined criteria
- **so that** submissions are ranked fairly based on multiple quality dimensions

#### Acceptance Criteria:

- **Scenario:** Judge scores a submission
- **Given:** The hackathon is in `Judging(3)` status
- **and Given:** I am assigned as `judge1`
- **When:** I submit scores for hacker1's submission: Innovation=90, Technical Quality=85, Presentation=80
- **Then:** My scores are recorded, weighted score is calculated (90×0.4 + 85×0.35 + 80×0.25 = 85.75), and a `hackathon_scored(33)` feed event is emitted

**Journey ref:** J1 Steps 1.17, 1.18
