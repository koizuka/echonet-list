import socket
import time

IP = '192.168.0.255' # broadcast
#IP = '192.168.0.212' # WTY2001のIPアドレス
PORT = 3610
pkt = '1081 0000 0ef001 0ef001 62 01 d6 00' # get instance list

sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
sock.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
sock.setsockopt(socket.SOL_SOCKET, socket.SO_BROADCAST, 1)
sock.bind(('', PORT))
sock.sendto(bytes.fromhex(pkt), (IP, PORT))


sock.settimeout(5.0)  # 5秒間のタイムアウトを設定

start_time = time.time()
while time.time() - start_time < 5.0:
    try:
        msg, addr = sock.recvfrom(1500)
        print(addr, msg.hex())
    except socket.timeout:
        break  # タイムアウトしたらループを抜ける

#msg, addr = sock.recvfrom(1500)
#print(addr, msg.hex())
sock.close()
