module github.com/josh/smtp2webhook

go 1.16

require (
	// bump: go-smtp /github.com\/emersion\/go-smtp v(.*)/ git:https://github.com/emersion/go-smtp|^0
	// bump: go-smtp command go get -d github.com/emersion/go-smtp@v$LATEST && go mod tidy
	github.com/emersion/go-smtp v0.14.0
	github.com/namsral/flag v1.7.4-pre
)
