package esphome_apiclient

//go:generate protoc --go_out=pb --go_opt=paths=source_relative --go_opt=Mapi.proto=github.com/richard87/esphome-apiclient/pb --go_opt=Mapi_options.proto=github.com/richard87/esphome-apiclient/pb api.proto api_options.proto
