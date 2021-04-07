package storage
import (
        "fmt"
        "os"
        "github.com/aws/aws-sdk-go/aws"
        "github.com/aws/aws-sdk-go/aws/credentials"
        "github.com/aws/aws-sdk-go/aws/session"
        "github.com/aws/aws-sdk-go/service/s3/s3manager"
)

var AccessKeyID string
var SecretAccessKey string
var MyRegion string

func GetEnvWithKey(key string) string {
 return os.Getenv(key)
}

func ConnectAws() *session.Session {
        AccessKeyID = GetEnvWithKey("AWS_ACCESS_KEY_ID")
        SecretAccessKey = GetEnvWithKey("AWS_SECRET_ACCESS_KEY")
        MyRegion = GetEnvWithKey("AWS_REGION")

        sess, err := session.NewSession(
               &aws.Config{
                      Region: aws.String(MyRegion),
                      Credentials: credentials.NewStaticCredentials(
                      AccessKeyID,
                      SecretAccessKey,
                      "", // a token will be created when the session it's used.
               ),
       })

       if err != nil {
             panic(err)
       }

        return sess
}

func UploadData (session *session.Session) {
        uploader := s3manager.NewUploader(session)
}

func main() {
  sess := ConnectAws()
  fmt.Println(sess)
  UploadData(sess)
}
