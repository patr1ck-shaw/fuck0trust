import argparse
import ctypes
import getpass
import hashlib
import json
import os
import platform
import subprocess
import sys
import uuid
from pathlib import Path
from tkinter import END, DISABLED, NORMAL, Button, Entry, Label, StringVar, Text, Tk, messagebox
from tkinter.scrolledtext import ScrolledText

import requests


APP_NAME = "Fuck0TrustApprovalClient"
TASK_NAME = "Fuck0Trust_Status_Check"
SERVICE_NAME = "WFPRedirect"
CONFIG_DIR = Path(os.environ.get("PROGRAMDATA", str(Path.home()))) / APP_NAME
CONFIG_FILE = CONFIG_DIR / "config.json"


def is_admin() -> bool:
    try:
        return bool(ctypes.windll.shell32.IsUserAnAdmin())
    except Exception:
        return False


def machine_guid() -> str:
    try:
        output = subprocess.check_output(
            ["reg", "query", r"HKLM\SOFTWARE\Microsoft\Cryptography", "/v", "MachineGuid"],
            text=True,
            stderr=subprocess.DEVNULL,
            encoding="utf-8",
            errors="ignore",
        )
        for line in output.splitlines():
            if "MachineGuid" in line:
                return line.split()[-1].strip()
    except Exception:
        pass
    return str(uuid.getnode())


def device_id() -> str:
    raw = "|".join(
        [
            platform.node(),
            platform.system(),
            platform.machine(),
            machine_guid(),
        ]
    )
    return hashlib.sha256(raw.encode("utf-8", errors="ignore")).hexdigest()


def load_config() -> dict:
    if CONFIG_FILE.exists():
        return json.loads(CONFIG_FILE.read_text(encoding="utf-8"))
    return {}


def save_config(config: dict) -> None:
    CONFIG_DIR.mkdir(parents=True, exist_ok=True)
    CONFIG_FILE.write_text(json.dumps(config, ensure_ascii=False, indent=2), encoding="utf-8")


def api_base_from_args(args: argparse.Namespace) -> str:
    if args.api:
        config = load_config()
        config["api"] = args.api.rstrip("/")
        save_config(config)
        return config["api"]
    config = load_config()
    api = config.get("api") or os.environ.get("APPROVAL_API")
    if not api:
        raise SystemExit("请先通过 --api 设置 Cloudflare Worker 地址，例如：client.exe --api https://xxx.workers.dev request")
    return str(api).rstrip("/")


def request_approval(args: argparse.Namespace) -> None:
    api = api_base_from_args(args)
    did = device_id()
    payload = {
        "deviceId": did,
        "hostname": platform.node(),
        "username": getpass.getuser(),
        "note": args.note or "",
    }
    resp = requests.post(f"{api}/api/request", json=payload, timeout=20)
    print_json(resp)
    print(f"\n设备 ID: {did}")
    print("已提交审批申请，请联系管理员审批。")


def request_approval_data(api: str, note: str = "") -> dict:
    did = device_id()
    payload = {
        "deviceId": did,
        "hostname": platform.node(),
        "username": getpass.getuser(),
        "note": note or "",
    }
    resp = requests.post(f"{api.rstrip('/')}/api/request", json=payload, timeout=20)
    data = resp.json()
    resp.raise_for_status()
    return data


def status(args: argparse.Namespace) -> bool:
    api = api_base_from_args(args)
    did = device_id()
    resp = requests.get(f"{api}/api/status", params={"deviceId": did}, timeout=20)
    data = print_json(resp)
    approved = bool(data.get("approved")) if isinstance(data, dict) else False
    print(f"\n设备 ID: {did}")
    print("审批状态：已通过" if approved else "审批状态：未通过/待审批")
    return approved


def status_data(api: str) -> dict:
    did = device_id()
    resp = requests.get(f"{api.rstrip('/')}/api/status", params={"deviceId": did}, timeout=20)
    data = resp.json()
    resp.raise_for_status()
    return data


def ensure_approved(args: argparse.Namespace) -> None:
    if not status(args):
        raise SystemExit("当前设备未审批通过，不能执行受控功能。")


def query_wfp_status() -> None:
    print(f"[INFO] 当前 {SERVICE_NAME} 状态：")
    subprocess.run(["sc", "query", SERVICE_NAME], check=False)
    print("\n说明：本工具仅查询状态，不会停止或禁用安全/零信任驱动。")


def run_once(args: argparse.Namespace) -> None:
    ensure_approved(args)
    query_wfp_status()


def current_exe_path() -> str:
    return str(Path(sys.executable if getattr(sys, "frozen", False) else __file__).resolve())


def install_task(args: argparse.Namespace) -> None:
    ensure_approved(args)
    if not is_admin():
        raise SystemExit("写入系统计划任务需要管理员权限，请右键以管理员身份运行。")

    exe = current_exe_path()
    api = api_base_from_args(args)
    command = f'"{exe}" --api "{api}" run'
    schtasks_cmd = [
        "schtasks",
        "/Create",
        "/TN",
        TASK_NAME,
        "/TR",
        command,
        "/SC",
        "MINUTE",
        "/MO",
        "4",
        "/RL",
        "HIGHEST",
        "/F",
    ]
    subprocess.run(schtasks_cmd, check=True)
    print(f"计划任务已创建/更新：{TASK_NAME}，每 4 分钟执行一次状态检查。")


def remove_task(_: argparse.Namespace) -> None:
    if not is_admin():
        raise SystemExit("删除系统计划任务需要管理员权限，请右键以管理员身份运行。")
    subprocess.run(["schtasks", "/Delete", "/TN", TASK_NAME, "/F"], check=False)
    print(f"计划任务已删除：{TASK_NAME}")


def print_json(resp: requests.Response):
    try:
        data = resp.json()
    except Exception:
        print(resp.text)
        resp.raise_for_status()
        return None
    print(json.dumps(data, ensure_ascii=False, indent=2))
    resp.raise_for_status()
    return data


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description="设备审批客户端（安全版）")
    parser.add_argument("--api", help="Cloudflare Worker 地址，例如 https://xxx.workers.dev")
    sub = parser.add_subparsers(dest="command", required=True)

    p_request = sub.add_parser("request", help="提交当前设备审批申请")
    p_request.add_argument("--note", help="申请备注")
    p_request.set_defaults(func=request_approval)

    p_status = sub.add_parser("status", help="查询当前设备审批状态")
    p_status.set_defaults(func=status)

    p_run = sub.add_parser("run", help="审批通过后执行一次受控功能：查询 WFPRedirect 状态")
    p_run.set_defaults(func=run_once)

    p_install = sub.add_parser("install-task", help="审批通过后安装每 4 分钟执行一次的状态检查计划任务")
    p_install.set_defaults(func=install_task)

    p_remove = sub.add_parser("remove-task", help="删除计划任务")
    p_remove.set_defaults(func=remove_task)
    return parser


def launch_gui() -> None:
    root = Tk()
    root.title("Fuck0Trust 设备审批客户端")
    root.geometry("760x560")

    config = load_config()
    api_var = StringVar(value=str(config.get("api", os.environ.get("APPROVAL_API", ""))))
    note_var = StringVar(value="")
    did = device_id()

    Label(root, text="Cloudflare Worker API 地址：").pack(anchor="w", padx=12, pady=(12, 2))
    api_entry = Entry(root, textvariable=api_var, width=100)
    api_entry.pack(fill="x", padx=12)

    Label(root, text="申请备注：").pack(anchor="w", padx=12, pady=(10, 2))
    note_entry = Entry(root, textvariable=note_var, width=100)
    note_entry.pack(fill="x", padx=12)

    Label(root, text=f"当前设备 ID：{did}").pack(anchor="w", padx=12, pady=(10, 2))

    output: Text = ScrolledText(root, height=18)
    output.pack(fill="both", expand=True, padx=12, pady=12)

    def append(text: str) -> None:
        output.configure(state=NORMAL)
        output.insert(END, text + "\n")
        output.see(END)
        output.configure(state=DISABLED)

    def get_api() -> str:
        api = api_var.get().strip().rstrip("/")
        if not api:
            raise ValueError("请先填写 Cloudflare Worker API 地址，例如：https://xxx.workers.dev")
        config = load_config()
        config["api"] = api
        save_config(config)
        return api

    def run_action(name: str, action) -> None:
        append(f"\n>>> {name}")
        try:
            result = action()
            if result is not None:
                append(json.dumps(result, ensure_ascii=False, indent=2))
        except Exception as exc:
            append(f"错误：{exc}")
            messagebox.showerror("执行失败", str(exc))

    def gui_request() -> dict:
        api = get_api()
        data = request_approval_data(api, note_var.get())
        append(f"设备 ID: {did}")
        append("已提交审批申请，请联系管理员审批。")
        return data

    def gui_status() -> dict:
        api = get_api()
        data = status_data(api)
        append(f"审批状态：{'已通过' if data.get('approved') else '未通过/待审批'}")
        return data

    def gui_run() -> None:
        args = argparse.Namespace(api=get_api())
        run_once(args)
        append("受控功能已执行：已查询 WFPRedirect 状态。")

    def gui_install_task() -> None:
        args = argparse.Namespace(api=get_api())
        install_task(args)
        append(f"计划任务已创建/更新：{TASK_NAME}，每 4 分钟执行一次。")

    def gui_remove_task() -> None:
        args = argparse.Namespace(api=None)
        remove_task(args)
        append(f"计划任务已删除：{TASK_NAME}")

    button_frame = root
    Button(button_frame, text="提交审批", command=lambda: run_action("提交审批", gui_request)).pack(side="left", padx=(12, 4), pady=6)
    Button(button_frame, text="查询状态", command=lambda: run_action("查询状态", gui_status)).pack(side="left", padx=4, pady=6)
    Button(button_frame, text="执行一次", command=lambda: run_action("执行一次", gui_run)).pack(side="left", padx=4, pady=6)
    Button(button_frame, text="安装4分钟计划任务", command=lambda: run_action("安装计划任务", gui_install_task)).pack(side="left", padx=4, pady=6)
    Button(button_frame, text="删除计划任务", command=lambda: run_action("删除计划任务", gui_remove_task)).pack(side="left", padx=4, pady=6)

    append("欢迎使用设备审批客户端。")
    append("首次使用请填写 Worker API 地址并点击“提交审批”。")
    append("注意：本工具仅查询状态，不会停止或禁用安全/零信任驱动。")
    root.mainloop()


def main() -> None:
    if len(sys.argv) == 1:
        launch_gui()
        return
    parser = build_parser()
    args = parser.parse_args()
    args.func(args)


if __name__ == "__main__":
    main()
