module github.com/sdslabs/beastv4

require (
	github.com/BurntSushi/toml v1.4.0
	github.com/alecthomas/template v0.0.0-20190718012654-fb15b899a751
	github.com/araddon/dateparse v0.0.0-20210429162001-6b43995a97de
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/docker/docker v1.13.1
	github.com/docker/go-connections v0.4.0
	github.com/gin-contrib/cors v1.3.1
	github.com/gin-contrib/static v0.0.0-20200916080430-d45d9a37d28e
	github.com/gin-gonic/gin v1.7.0
	github.com/go-sql-driver/mysql v1.4.0
	github.com/golang/protobuf v1.3.3
	github.com/jinzhu/gorm v1.9.1
	github.com/mohae/struct2csv v0.0.0-20151122200941-e72239694eae
	github.com/olekukonko/tablewriter v0.0.5
	github.com/sirupsen/logrus v1.0.6
	github.com/spf13/cobra v0.0.3
	github.com/swaggo/gin-swagger v1.0.0
	github.com/swaggo/swag v1.16.4
	golang.org/x/crypto v0.29.0
	golang.org/x/net v0.31.0
	google.golang.org/grpc v1.19.0
	gopkg.in/src-d/go-git.v4 v4.7.0
	gorm.io/driver/sqlite v1.1.3
	gorm.io/gorm v1.20.6
)

require (
	cloud.google.com/go v0.28.0 // indirect
	github.com/KyleBanks/depth v1.2.1 // indirect
	github.com/Microsoft/go-winio v0.4.11 // indirect
	github.com/alcortesm/tgz v0.0.0-20161220082320-9c5fe88206d7 // indirect
	github.com/anmitsu/go-shlex v0.0.0-20161002113705-648efa622239 // indirect
	github.com/cpuguy83/go-md2man v1.0.10 // indirect
	github.com/denisenkom/go-mssqldb v0.0.0-20180901172138-1eb28afdf9b6 // indirect
	github.com/docker/distribution v2.6.2+incompatible // indirect
	github.com/docker/go-units v0.3.3 // indirect
	github.com/emirpasic/gods v1.12.0 // indirect
	github.com/erikstmartin/go-testdb v0.0.0-20160219214506-8d10e4a1bae5 // indirect
	github.com/gin-contrib/sse v0.1.0 // indirect
	github.com/gliderlabs/ssh v0.1.1 // indirect
	github.com/go-openapi/jsonpointer v0.21.0 // indirect
	github.com/go-openapi/jsonreference v0.21.0 // indirect
	github.com/go-openapi/spec v0.21.0 // indirect
	github.com/go-openapi/swag v0.23.0 // indirect
	github.com/go-playground/locales v0.13.0 // indirect
	github.com/go-playground/universal-translator v0.17.0 // indirect
	github.com/go-playground/validator/v10 v10.4.1 // indirect
	github.com/inconshreveable/mousetrap v1.0.0 // indirect
	github.com/jbenet/go-context v0.0.0-20150711004518-d14ea06fba99 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.1 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.9 // indirect
	github.com/kevinburke/ssh_config v0.0.0-20180830205328-81db2a75821e // indirect
	github.com/leodido/go-urn v1.2.0 // indirect
	github.com/lib/pq v1.0.0 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/mattn/go-isatty v0.0.12 // indirect
	github.com/mattn/go-runewidth v0.0.10 // indirect
	github.com/mattn/go-sqlite3 v1.14.3 // indirect
	github.com/mitchellh/go-homedir v1.0.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.1 // indirect
	github.com/pelletier/go-buffruneio v0.2.0 // indirect
	github.com/pkg/errors v0.8.0 // indirect
	github.com/rivo/uniseg v0.1.0 // indirect
	github.com/russross/blackfriday v1.5.2 // indirect
	github.com/sergi/go-diff v1.0.0 // indirect
	github.com/spf13/pflag v1.0.2 // indirect
	github.com/src-d/gcfg v1.3.0 // indirect
	github.com/stevvooe/resumable v0.0.0-20180830230917-22b14a53ba50 // indirect
	github.com/ugorji/go/codec v1.1.7 // indirect
	github.com/xanzy/ssh-agent v0.2.0 // indirect
	golang.org/x/sys v0.27.0 // indirect
	golang.org/x/term v0.26.0 // indirect
	golang.org/x/text v0.20.0 // indirect
	golang.org/x/tools v0.27.0 // indirect
	google.golang.org/appengine v1.2.0 // indirect
	google.golang.org/genproto v0.0.0-20180817151627-c66870c02cf8 // indirect
	gopkg.in/airbrake/gobrake.v2 v2.0.9 // indirect
	gopkg.in/gemnasium/logrus-airbrake-hook.v2 v2.1.2 // indirect
	gopkg.in/src-d/go-billy.v4 v4.3.0 // indirect
	gopkg.in/src-d/go-git-fixtures.v3 v3.3.0 // indirect
	gopkg.in/warnings.v0 v0.1.2 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/docker/docker v1.13.1 => github.com/docker/engine v1.13.1

go 1.22.0

toolchain go1.22.5
