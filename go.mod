module github.com/go-sample-project

go 1.16

require (
	github.com/gorilla/mux v1.8.1
	github.com/mattn/go-sqlite3 v1.14.22
	github.com/sirupsen/logrus v1.9.3
	github.developer.allianz.io/global-blockchain-centre-of-competence/ics-lib-go v0.0.0-20240229044653-a91fc257f7ab
	github.developer.allianz.io/global-blockchain-centre-of-competence/ics-service-foreign-claim-api v0.0.0-20240222071839-6c31226e9995
	golang.org/x/crypto v0.0.0-20211108221036-ceb1ce70b4fa
	google.golang.org/grpc v1.54.1
)

replace github.com/luthersystems/lutherauth-sdk-go v0.0.2 => github.developer.allianz.io/global-blockchain-centre-of-competence/lutherauth-sdk-go v0.0.2
