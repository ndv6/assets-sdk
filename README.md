# assets-sdk
Purpose as centralise SDK to handle things related to assets such as image upload, etc


## File
Upload and delete file from Azure storage


### Usages


#### func Upload
    func Upload(ctx context.Context, filePath string, buffBytes []byte) (string, error)
Upload file to storage

Example:

`file := file.Upload(ctx, "/file/image.img", buffBytes)`


#### func Delete
    func Delete(ctx context.Context, filePath string) (string, error)
Delete file from storage

Example:

`file := file.Delete(ctx, "/file/image.img")`

