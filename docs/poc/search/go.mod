module sign

go 1.24.5

require (
	github.com/opencontainers/go-digest v1.0.0
	github.com/opencontainers/image-spec v1.1.1
	oras.land/oras-go/v2 v2.5.0
)

require golang.org/x/sync v0.15.0 // indirect

replace github.com/agntcy/dir/utils => ../../utils
