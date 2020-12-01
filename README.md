# gitlab-mr-reminder

Simple script sending slack notifications for open merge requests for members of a specific group.

## configuration
Using environment variables:

GITLAB_DOMAIN: gitlab domain (mandatory)
GITLAB_TOKENL gitlab personal access token (mandatory)
GITLAB_GROUP_NAME: gitlab group name (mandatory)
GITLAB_GROUP_MEMBER_LEVEL: group member access level (see https://docs.gitlab.com/ee/api/access_requests.html) (optional)
GITLAB_MR_AGE: gitlab merge request age in hours (optional)
SLACK_WEBHOOK: slack webhook url (mandatory)
SLACK_CHANNEL: slack channel (optional)