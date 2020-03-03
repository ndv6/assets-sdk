# assets-sdk
Purpose as centralise SDK to handle things related to assets such as image upload, etc

## Usages

### func Upload
    func Upload(ctx context.Context, filePath string, buffBytes []byte) (string, error)
Upload file to storage

### func Delete
    func Delete(ctx context.Context, filePath string) (string, error)
Delete file from storage
    
### func GetBlobURL
    func GetBlobURL(fileName string, withSignature bool) string
Get full URL
    

