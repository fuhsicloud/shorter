/**
 * @Time : 19/11/2019 10:14 AM * @Author:ygqbasic@gmail.com * @File:repository * @Software:VsCode */

package service

type Repository interface {
	Find(code string)(redirect * Redirect, err error)
	Store(redirect * Redirect)error
	Exists(has string)(exists bool, err error)
}
