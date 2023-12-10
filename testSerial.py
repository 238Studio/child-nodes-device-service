# 我需要一个串口测试代码
# 用于测试串口的发送和接收
import serial
import time

def StartListen(portID):
    print("StartListen")
    ser = serial.Serial(portID, 115200, timeout=0.5)
    port=0
    while True:
        data = ser.read(512)
        print(data)
        if data != b'':
            port=int.from_bytes(data, byteorder='little', signed=False)
        ser.write('hello'.encode('utf-8'))
        time.sleep(0.5)
StartListen("COM6")