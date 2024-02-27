# 需求方（采购方）py文件

# _*_ coding:utf-8 _*_

import json
import socket
import time
# 因为服务端是TCP协议，所以我们直接使用socket

def checkConnection(note):
    request = {
        "id":0,
        "params":[note],
        "method":"GYClass.CheckConnection"
    }
    client = socket.create_connection(("localhost", 5100))
    client.sendall(json.dumps(request).encode())
    # 获取服务器返回的数据
    rsp = client.recv(1024) # 接收到的数据是二进制文本
    rsp = json.loads(rsp.decode())
    print(rsp)
    return rsp
    
def createNewAccount(note):
    request = {
        "id":0,
        "params":[note],
        "method":"GYClass.CreateNewAccount"
    }
    client = socket.create_connection(("localhost", 5100))
    client.sendall(json.dumps(request).encode())
    # 获取服务器返回的数据
    rsp = client.recv(1024) # 接收到的数据是二进制文本
    rsp = json.loads(rsp.decode())
    print(rsp)
    return rsp

def createSupply(note):
    request = {
        "id":0,
        "params":[note],
        "method":"GYClass.CreateSupply"
    }
    client = socket.create_connection(("localhost", 5100))
    client.sendall(json.dumps(request).encode())
    # 获取服务器返回的数据
    rsp = client.recv(1024) # 接收到的数据是二进制文本
    rsp = json.loads(rsp.decode())
    print(rsp)
    return rsp

def uploadPurchaseForm(note):
    request = {
        "id":0,
        "params":[note],
        "method":"GYClass.UploadPurchaseForm"
    }
    client = socket.create_connection(("localhost", 5100))
    client.sendall(json.dumps(request).encode())
    # 获取服务器返回的数据
    rsp = client.recv(1024) # 接收到的数据是二进制文本
    rsp = json.loads(rsp.decode())
    print(rsp)
    return rsp

def uploadReceiptForm(note):
    request = {
        "id":0,
        "params":[note],
        "method":"GYClass.UploadReceiptForm"
    }
    client = socket.create_connection(("localhost", 5100))
    client.sendall(json.dumps(request).encode())
    # 获取服务器返回的数据
    rsp = client.recv(1024) # 接收到的数据是二进制文本
    rsp = json.loads(rsp.decode())
    print(rsp)
    return rsp

def acceptPurchaseNote(note):
    request = {
        "id":0,
        "params":[note],
        "method":"GYClass.AcceptPurchaseNote"
    }
    client = socket.create_connection(("localhost", 5100))
    client.sendall(json.dumps(request).encode())
    # 获取服务器返回的数据
    rsp = client.recv(1024) # 接收到的数据是二进制文本
    rsp = json.loads(rsp.decode())
    print(rsp)
    return rsp

def acceptPayment(note):
    request = {
        "id":0,
        "params":[note],
        "method":"GYClass.AcceptPayment"
    }
    client = socket.create_connection(("localhost", 5100))
    client.sendall(json.dumps(request).encode())
    # 获取服务器返回的数据
    rsp = client.recv(1024) # 接收到的数据是二进制文本
    rsp = json.loads(rsp.decode())
    print(rsp)
    return rsp

def acceptSendGoods(note):
    request = {
        "id":0,
        "params":[note],
        "method":"GYClass.AcceptSendGoods"
    }
    client = socket.create_connection(("localhost", 5100))
    client.sendall(json.dumps(request).encode())
    # 获取服务器返回的数据
    rsp = client.recv(1024) # 接收到的数据是二进制文本
    rsp = json.loads(rsp.decode())
    print(rsp)
    return rsp
    
def acceptReceive(note):
    request = {
        "id":0,
        "params":[note],
        "method":"GYClass.AcceptReceive"
    }
    client = socket.create_connection(("localhost", 5100))
    client.sendall(json.dumps(request).encode())
    # 获取服务器返回的数据
    rsp = client.recv(1024) # 接收到的数据是二进制文本
    rsp = json.loads(rsp.decode())
    print(rsp)
    return rsp

def queryProcessNote(note):
    request = {
        "id":0,
        "params":[note],
        "method":"GYClass.QueryProcessNote"
    }
    client = socket.create_connection(("localhost", 5100))
    client.sendall(json.dumps(request).encode())
    # 获取服务器返回的数据
    rsp = client.recv(1024) # 接收到的数据是二进制文本
    rsp = json.loads(rsp.decode())
    print(rsp)
    return rsp

def queryPurchaseNote(note):
    request = {
        "id":0,
        "params":[note],
        "method":"GYClass.QueryPurchaseNote"
    }
    client = socket.create_connection(("localhost", 5100))
    client.sendall(json.dumps(request).encode())
    # 获取服务器返回的数据
    rsp = client.recv(1024) # 接收到的数据是二进制文本
    rsp = json.loads(rsp.decode())
    print(rsp)
    return rsp

def queryReceiptNote(note):
    request = {
        "id":0,
        "params":[note],
        "method":"GYClass.QueryReceiptNote"
    }
    client = socket.create_connection(("localhost", 5100))
    client.sendall(json.dumps(request).encode())
    # 获取服务器返回的数据
    rsp = client.recv(1024) # 接收到的数据是二进制文本
    rsp = json.loads(rsp.decode())
    print(rsp)
    return rsp


def JRcreateFinance(note):
    request = {
        "id":0,
        "params":[note],
        "method":"JRClass.CreateFinance"
    }
    client = socket.create_connection(("localhost", 5100))
    client.sendall(json.dumps(request).encode())
    # 获取服务器返回的数据
    rsp = client.recv(1024) # 接收到的数据是二进制文本
    rsp = json.loads(rsp.decode())
    print(rsp)
    return rsp

def JRuploadLoanForm(note):
    request = {
        "id":0,
        "params":[note],
        "method":"JRClass.UploadLoanForm"
    }
    client = socket.create_connection(("localhost", 5100))
    client.sendall(json.dumps(request).encode())
    # 获取服务器返回的数据
    rsp = client.recv(1024) # 接收到的数据是二进制文本
    rsp = json.loads(rsp.decode())
    print(rsp)
    return rsp

def JRuploadPayForm(note):
    request = {
        "id":0,
        "params":[note],
        "method":"JRClass.UploadPayForm"
    }
    client = socket.create_connection(("localhost", 5100))
    client.sendall(json.dumps(request).encode())
    # 获取服务器返回的数据
    rsp = client.recv(1024) # 接收到的数据是二进制文本
    rsp = json.loads(rsp.decode())
    print(rsp)
    return rsp

def JRuploadRepayForm(note):
    request = {
        "id":0,
        "params":[note],
        "method":"JRClass.UploadRepayForm"
    }
    client = socket.create_connection(("localhost", 5100))
    client.sendall(json.dumps(request).encode())
    # 获取服务器返回的数据
    rsp = client.recv(1024) # 接收到的数据是二进制文本
    rsp = json.loads(rsp.decode())
    print(rsp)
    return rsp

def JRqueryProcessNote(note):
    request = {
        "id":0,
        "params":[note],
        "method":"JRClass.QueryProcessNote"
    }
    client = socket.create_connection(("localhost", 5100))
    client.sendall(json.dumps(request).encode())
    # 获取服务器返回的数据
    rsp = client.recv(1024) # 接收到的数据是二进制文本
    rsp = json.loads(rsp.decode())
    print(rsp)
    return rsp

def JRqueryLoanNote(note):
    request = {
        "id":0,
        "params":[note],
        "method":"JRClass.QueryLoanNote"
    }
    client = socket.create_connection(("localhost", 5100))
    client.sendall(json.dumps(request).encode())
    # 获取服务器返回的数据
    rsp = client.recv(1024) # 接收到的数据是二进制文本
    rsp = json.loads(rsp.decode())
    print(rsp)
    return rsp

def JRqueryPayNote(note):
    request = {
        "id":0,
        "params":[note],
        "method":"JRClass.QueryPayNote"
    }
    client = socket.create_connection(("localhost", 5100))
    client.sendall(json.dumps(request).encode())
    # 获取服务器返回的数据
    rsp = client.recv(1024) # 接收到的数据是二进制文本
    rsp = json.loads(rsp.decode())
    print(rsp)
    return rsp

def JRqueryRepayNote(note):
    request = {
        "id":0,
        "params":[note],
        "method":"JRClass.QueryRepayNote"
    }
    client = socket.create_connection(("localhost", 5100))
    client.sendall(json.dumps(request).encode())
    # 获取服务器返回的数据
    rsp = client.recv(1024) # 接收到的数据是二进制文本
    rsp = json.loads(rsp.decode())
    print(rsp)
    return rsp

def JRacceptCreditSideNote(note):
    request = {
        "id":0,
        "params":[note],
        "method":"JRClass.AcceptCreditSideNote"
    }
    client = socket.create_connection(("localhost", 5100))
    client.sendall(json.dumps(request).encode())
    # 获取服务器返回的数据
    rsp = client.recv(1024) # 接收到的数据是二进制文本
    rsp = json.loads(rsp.decode())
    print(rsp)
    return rsp

def JRacceptPayNote(note):
    request = {
        "id":0,
        "params":[note],
        "method":"JRClass.AcceptPayNote"
    }
    client = socket.create_connection(("localhost", 5100))
    client.sendall(json.dumps(request).encode())
    # 获取服务器返回的数据
    rsp = client.recv(1024) # 接收到的数据是二进制文本
    rsp = json.loads(rsp.decode())
    print(rsp)
    return rsp

def JRconfirmPayNote(note):
    request = {
        "id":0,
        "params":[note],
        "method":"JRClass.ConfirmPayNote"
    }
    client = socket.create_connection(("localhost", 5100))
    client.sendall(json.dumps(request).encode())
    # 获取服务器返回的数据
    rsp = client.recv(1024) # 接收到的数据是二进制文本
    rsp = json.loads(rsp.decode())
    print(rsp)
    return rsp

def JRacceptRepayNote(note):
    request = {
        "id":0,
        "params":[note],
        "method":"JRClass.AcceptRepayNote"
    }
    client = socket.create_connection(("localhost", 5100))
    client.sendall(json.dumps(request).encode())
    # 获取服务器返回的数据
    rsp = client.recv(1024) # 接收到的数据是二进制文本
    rsp = json.loads(rsp.decode())
    print(rsp)
    return rsp

def JRconfirmRepayNote(note):
    request = {
        "id":0,
        "params":[note],
        "method":"JRClass.ConfirmRepayNote"
    }
    client = socket.create_connection(("localhost", 5100))
    client.sendall(json.dumps(request).encode())
    # 获取服务器返回的数据
    rsp = client.recv(1024) # 接收到的数据是二进制文本
    rsp = json.loads(rsp.decode())
    print(rsp)
    return rsp



if __name__ == '__main__':


    #服务演示
    process_id = "hcnodiajcndoiabanondiancoidnnrionionronviro" #供应流程唯一id
   
    #测试通信功能是否正常  
    checkConnection("")
    
    #为接收方和供应商创建账号
    tmp = createNewAccount("supplyer_name")
    supplyer_addr = tmp["result"]
    time.sleep(3)
    tmp = createNewAccount("receiver_name")
    receiver_addr = tmp["result"]
    time.sleep(3)




    #创建一次供应流程
    request_data = receiver_addr+";"+supplyer_addr+";"+process_id+";"+"供应商"+";"+"接受商"+";"
    createSupply(request_data)
    time.sleep(3)



    #上传purchase表单
    request_data =   process_id+";"+ \
                    "Contract_identifier"  +";"+ \
                    "Buyer"+";"+ \
                    "Provider_name"+";"+ \
                    "Goods_name"+";"+ \
                    "Goods_type"+";"+ \
                    "Goods_specification"+";"+ \
                    "Goods_amount"+";"+ \
                    "Goods_unit"+";"+ \
                    "Goods_price"+";"+ \
                    "Goods_total_price"+";"+ \
                    "Due_and_amount"+";"+ \
                    "Sign_date"+";"+ \
                    receiver_addr +";"#上传purchase表单需要receiver_addr
    print(receiver_addr)
    uploadPurchaseForm(request_data)
    time.sleep(3)

    #上传receipt表单
    request_data =   process_id+";"+ \
                    "Contract_identifier"  +";"+ \
                    "Buyer"+";"+ \
                    "Provider_name"+";"+ \
                    "Goods_name"+";"+ \
                    "Goods_type"+";"+ \
                    "Goods_specification"+";"+ \
                    "Goods_amount"+";"+ \
                    "Goods_unit"+";"+ \
                    "Goods_price"+";"+ \
                    "Goods_total_price"+";"+ \
                    "Date"+";"+ \
                    supplyer_addr +";"#上传receipt表单需要supplyer_addr
    uploadReceiptForm(request_data)
    time.sleep(3)
    
    #签名 Credit_side_status
    request_data = process_id+";"+supplyer_addr+";"
    acceptPurchaseNote(request_data)
    time.sleep(3)    
    
    #签名 Pay_status
    request_data = process_id+";"+receiver_addr+";"
    acceptPayment(request_data)
    time.sleep(3)
    
    #签名 Delivery_status
    request_data = process_id +";"+supplyer_addr+";"
    acceptSendGoods(request_data)
    time.sleep(3)
    
    #签名 Receive_status
    request_data = process_id+";"+receiver_addr+";"
    acceptReceive(request_data)
    time.sleep(10)
    
    #查询 process note 
    request_data = process_id +";"+ receiver_addr+";"#查询参数也可以是supply_addr
    queryProcessNote(request_data)
    time.sleep(10)
    
    #查询 Purchase note
    request_data = process_id +";"+ receiver_addr+";"#查询参数也可以是supply_addr
    queryPurchaseNote(request_data)
    time.sleep(10)
    
    #查询 Receipt note
    request_data = process_id +";"+ receiver_addr+";"#查询参数也可以是supply_addr
    queryReceiptNote(request_data)

    

    ########################################################################


    #金融链流程

    #服务演示
    process_id = "cdcdcdcdcdcdacdasdcacd" #金融流程唯一id
   
    #测试通信功能是否正常  
    checkConnection("")
    
    #为借款人和贷款人创建账号
    tmp = createNewAccount("borrower_name")
    borrower_addr = tmp["result"]
    time.sleep(3)
    tmp = createNewAccount("debtor_name")
    debtor_addr = tmp["result"]
    time.sleep(3)

    #创建一次金融贷款流程
    request_data = debtor_addr+";"+borrower_addr+";"+process_id+";"+"debtor_name"+";"+"borrower_name" 
    JRcreateFinance(request_data)
    time.sleep(10)

    #上传贷款表单 loan
    request_data =     process_id               + ";" + \
                       "Serial_no               " + ";" + \
                       "Amount                  " + ";" + \
                       "Fin_amount              " + ";" + \
                       "Loan_amount             " + ";" + \
                       "Ware_no                 " + ";" + \
                       "Contract_no             " + ";" + \
                       "Mass_no                 " + ";" + \
                       "Pledge_type             " + ";" + \
                       "Debtor_name             " + ";" + \
                       "Credit_side_name        " + ";" + \
                       "Fin_start_date          " + ";" + \
                       "Fin_end_date            " + ";" + \
                       "Fin_days                " + ";" + \
                       "Service_price           " + ";" + \
                       "Fund_prod_name          " + ";" + \
                       "Fund_prod_int_rate      " + ";" + \
                       "Fund_prod_service_price " + ";" + \
                       "Fund_prod_period        " + ";" + \
                       "Payment_type            " + ";" + \
                       "Bank_card_no            " + ";" + \
                       "Bank_name               " + ";" + \
                       "Remark                  " + ";" + \
                        borrower_addr 
    JRuploadLoanForm(request_data)
    time.sleep(10)

    #上传支付表单
    request_data =  process_id    + ";" + \
                    "Dk_no       " + ";"+ \
                    "Dk_name     " + ";"+ \
                    "Dk_quota    " + ";"+ \
                    "Pay_account " + ";"+ \
                    "Credit_start" + ";"+ \
                    "Credit_end  " + ";"+ \
                    borrower_addr 	
    JRuploadPayForm(request_data)
    time.sleep(5)

    #上传偿还表单
    request_data = process_id    + ";" + \
                    "Repay_no   "+ ";" + \
                    "Repay_quota"+ ";" + \
                    "Repay_date "+ ";" + \
                    debtor_addr
    JRuploadRepayForm(request_data)
    time.sleep(5)
    
    #签名 Credit_side_status
    request_data = process_id +";"+borrower_addr+";"
    JRacceptCreditSideNote(request_data)
    time.sleep(3)

    #签名 Credit_pay
    request_data = process_id + ";" + borrower_addr+";"
    JRacceptPayNote(request_data)
    time.sleep(3)

    #签名 Debtor_confirm_shoukuan
    request_data = process_id + ";" + debtor_addr + ";"
    JRconfirmPayNote(request_data)
    time.sleep(3)
    
    #签名 Debtor_repay
    request_data = process_id + ";" + debtor_addr + ";"
    JRacceptRepayNote(request_data)
    time.sleep(3)

    #签名  Credit_confirm_huankuan
    request_data = process_id + ";" + borrower_addr+";"
    JRconfirmRepayNote(request_data)
    time.sleep(3)

     #查询 流程表单
    request_data = process_id + ";" + borrower_addr+";"
    JRqueryProcessNote(request_data)
    time.sleep(3)
   
    #查询 支付表单
    request_data = process_id + ";" + borrower_addr+";"
    JRqueryPayNote(request_data)
    time.sleep(3)

    #查询 偿还表单
    request_data = process_id + ";" + borrower_addr+";"
    JRqueryRepayNote(request_data)
    time.sleep(3)

    #查询 贷款表单
    request_data = process_id + ";" + borrower_addr+";"
    JRqueryLoanNote(request_data)
    time.sleep(3)
    


