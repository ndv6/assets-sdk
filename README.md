# assets-sdk
Purpose as centralise SDK to handle things related to assets such as image upload, etc




# File
Upload and delete file from Azure storage


## Usages
### Initialization
Set Azure configuration first

    func New(account, accessKey, rootURL, containerName, apiVersion string)

## Function
### func Upload
Upload file to storage

    func Upload(ctx context.Context, filePath string, buffBytes []byte) (string, error)

Example:

`file := file.Upload(ctx, "/file/image.img", buffBytes)`

### func Delete
Delete file from storage

    func Delete(ctx context.Context, filePath string) (string, error)

Example:

`file := file.Delete(ctx, "/file/image.img")`

