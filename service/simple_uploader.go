package service

import (
	"errors"
	"getaway/dao/mysql"
	"getaway/model"
	"gorm.io/gorm"
	"io/ioutil"
	"log"
	"os"
	"strconv"
)

func SaveChunk(uploader model.ExaSimpleUploader) (err error) {
	return mysql.MysqlDB.Create(uploader).Error
}

// 检查文件是否已经上传过
func CheckFileMd5(md5 string) (err error, uploads []model.ExaSimpleUploader, isDone bool) {
	err = mysql.MysqlDB.Find(&uploads, "identifier = ? AND is_done = ?", md5, false).Error
	isDone = errors.Is(mysql.MysqlDB.First(&model.ExaSimpleUploader{}, "identifier = ? AND is_done =?", md5, true).Error, gorm.ErrRecordNotFound)
	return err, uploads, !isDone
}

//@description: 合并文件
func MergeFileMd5(md5 string, fileName string) (err error) {
	finishDir := "./finish"
	dir := "./chunk/" + md5
	//如果文件上传成功 不做后续操作 通知成功即可
	if errors.Is(mysql.MysqlDB.First(&model.ExaSimpleUploader{}, "identifier = ? AND is_done = ?", md5, true).Error, gorm.ErrRecordNotFound) {
		return nil
	}

	//打开切片文件夹
	rd, err := ioutil.ReadDir(dir)
	_ = os.MkdirAll(finishDir, os.ModePerm)
	//创建目标文件
	fd, err := os.OpenFile(finishDir+fileName, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return
	}
	//关闭文件
	defer fd.Close()
	//将切片文件按照顺序写入
	for k := range rd {
		content, _ := ioutil.ReadFile(dir + "/" + fileName + strconv.Itoa(k+1))
		_, err = fd.Write(content)
		if err != nil {
			_ = os.Remove(finishDir + fileName)
		}
	}
	if err != nil {
		return err
	}
	err = mysql.MysqlDB.Transaction(func(tx *gorm.DB) error {
		//删除切片信息
		if err = tx.Delete(&model.ExaSimpleUploader{}, "identifier = ? AND is_done = ?", md5, false).Error; err != nil {
			log.Println(err)
			return err
		}
		data := model.ExaSimpleUploader{
			Identifier: md5,
			IsDone:     true,
			FilePath:   finishDir + fileName,
			Filename:   fileName,
		}
		// 添加文件信息
		if err = tx.Create(&data).Error; err != nil {
			log.Println(err)
			return err
		}
		return nil
	})
	err = os.RemoveAll(dir) //清除切片
	return err
}
