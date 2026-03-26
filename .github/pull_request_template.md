# 🔁 Pull Request Template – BharatMLStack
> Please fill out the following sections to help us review your changes efficiently.

## Context:
Give a brief overview of the motivation behind this change. Include any relevant discussion links (Slack, documents, tickets, etc.) that help reviewers understand the background and the issue being addressed.


## Describe your changes:
Mention the changes made in the codebase.

## Testing:
Please describe how you tested the code. If manual tests were performed - please explain how. If automatic tests were added or existing ones cover the change - please explain how did you run them.

## Monitoring:
Explain how this change will be tracked after deployment. Indicate whether current dashboards, alerts, and logs are enough, or if additional instrumentation is required.

## Rollback plan 
Explain rollback plan in case of issues.

## Checklist before requesting a review
- [ ] I have reviewed my own changes?
- [ ] Relevant or critical functionality is covered by tests?
- [ ] Monitoring needs have been evaluated?
- [ ] Any necessary documentation updates have been considered?

## 📂 Modules Affected
<!-- Tick all that apply -->
- [ ] `horizon` (Real-time systems / networking)
- [ ] `online-feature-store` (Feature serving infra)
- [ ] `trufflebox-ui` (Admin panel / UI)
- [ ] `infra` (Docker, CI/CD, GCP/AWS setup)
- [ ] `docs` (Documentation updates)
- [ ] Other: `___________`

---

## ✅ Type of Change

- [ ] Feature addition
- [ ] Bug fix
- [ ] Infra / build system change
- [ ] Performance improvement
- [ ] Refactor
- [ ] Documentation
- [ ] Other: `___________`

---

## 📊 Benchmark / Metrics (if applicable)

<!-- Share perf impact (latency, throughput, mem usage, etc). Mention method of measurement -->