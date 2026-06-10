import argparse
import ctypes
import getpass
import hashlib
import json
import os
import platform
import subprocess
import sys
import time
import uuid
from pathlib import Path
from tkinter import Button, Entry, Frame, Label, StringVar, Tk, messagebox

import requests


APP_NAME = "Fuck0TrustApprovalClient"
TASK_NAME = "Fuck0Trust_Status_Check"
SERVICE_NAME = "WFPRedirect"
API_BASE = "https://0.cn01.eu.cc"
REQUEST_INTERVAL_SECONDS = 24 * 60 * 60
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


def now_ts() -> int:
    return int(time.time())


def api_base_from_args(args: argparse.Namespace) -> str:
    return API_BASE


def approval_cache_key() -> str:
    return f"approval:{device_id()}"


def request_cache_key() -> str:
    return f"request:{device_id()}"


def is_locally_approved() -> bool:
    cached = load_config().get(approval_cache_key())
    return bool(isinstance(cached, dict) and cached.get("approved") is True)


def save_local_approval(record: dict | None = None) -> None:
    config = load_config()
    config[approval_cache_key()] = {
        "approved": True,
        "deviceId": device_id(),
        "approvedAt": now_ts(),
        "record": record or {},
    }
    save_config(config)


def clear_local_approval() -> None:
    config = load_config()
    config.pop(approval_cache_key(), None)
    save_config(config)


def mark_request_submitted() -> None:
    config = load_config()
    config[request_cache_key()] = {"submittedAt": now_ts(), "deviceId": device_id()}
    save_config(config)


def seconds_until_next_request() -> int:
    cached = load_config().get(request_cache_key())
    if not isinstance(cached, dict):
        return 0
    elapsed = now_ts() - int(cached.get("submittedAt") or 0)
    return max(0, REQUEST_INTERVAL_SECONDS - elapsed)


def format_duration(seconds: int) -> str:
    hours = seconds // 3600
    minutes = (seconds % 3600) // 60
    if hours > 0:
        return f"{hours}小时{minutes}分钟"
    return f"{max(1, minutes)}分钟"


def check_api_reachable(timeout: int = 8) -> bool:
    resp = requests.get(f"{API_BASE}/health", timeout=timeout)
    resp.raise_for_status()
    return True


def refresh_approval_from_api(timeout: int = 10) -> dict:
    data = status_data(API_BASE, timeout=timeout)
    if data.get("approved"):
        save_local_approval(data.get("record") or data)
    else:
        clear_local_approval()
    return data


def request_approval(args: argparse.Namespace) -> None:
    api = api_base_from_args(args)
    remaining = seconds_until_next_request()
    if remaining > 0:
        raise SystemExit(f"同一设备 24 小时内只允许提交一次审批，请 {format_duration(remaining)} 后再试。")
    did = device_id()
    payload = {
        "deviceId": did,
        "hostname": platform.node(),
        "username": getpass.getuser(),
        "note": args.note or "",
    }
    resp = requests.post(f"{api}/api/request", json=payload, timeout=20)
    print_json(resp)
    mark_request_submitted()
    print(f"\n设备 ID: {did}")
    print("已提交审批申请，请联系管理员审批。")


def request_approval_data(api: str, note: str = "") -> dict:
    remaining = seconds_until_next_request()
    if remaining > 0:
        raise RuntimeError(f"同一设备 24 小时内只允许提交一次审批，请 {format_duration(remaining)} 后再试。")
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
    mark_request_submitted()
    return data


def status(args: argparse.Namespace) -> bool:
    did = device_id()
    data = refresh_approval_from_api(timeout=20)
    print(json.dumps(data, ensure_ascii=False, indent=2))
    approved = bool(data.get("approved")) if isinstance(data, dict) else False
    print(f"\n设备 ID: {did}")
    print("审批状态：已通过" if approved else "审批状态：未通过/待审批")
    return approved


def status_data(api: str, timeout: int = 20) -> dict:
    did = device_id()
    resp = requests.get(f"{api.rstrip('/')}/api/status", params={"deviceId": did}, timeout=timeout)
    data = resp.json()
    resp.raise_for_status()
    return data


def ensure_approved(args: argparse.Namespace) -> None:
    if not is_locally_approved():
        raise SystemExit("当前设备未审批通过，不能执行受控功能。请先打开客户端联网完成审批状态同步。")


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
    command = f'"{exe}" run'
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
    root.title("fuck0trust")
    root.geometry("560x420")
    root.resizable(False, False)

    note_var = StringVar(value="")
    status_var = StringVar(value="当前设备审批状态：检测中")
    did = device_id()

    root.configure(bg="#f6f7fb")
    header = Frame(root, bg="#2563eb", height=82)
    header.pack(fill="x")
    header.pack_propagate(False)
    Label(header, text="fuck0trust", bg="#2563eb", fg="white", font=("Microsoft YaHei", 24, "bold")).pack(expand=True)

    card = Frame(root, bg="white", padx=22, pady=18)
    card.pack(fill="both", expand=True, padx=22, pady=18)

    status_label = Label(card, textvariable=status_var, bg="white", fg="#334155", font=("Microsoft YaHei", 13, "bold"))
    status_label.pack(anchor="w", pady=(0, 12))
    Label(card, text="设备 ID：" + did[:16] + "..." + did[-8:], bg="white", fg="#64748b", font=("Microsoft YaHei", 9)).pack(anchor="w", pady=(0, 16))

    Label(card, text="申请备注（可选）：", bg="white", fg="#334155").pack(anchor="w")
    note_entry = Entry(card, textvariable=note_var, width=58)
    note_entry.pack(fill="x", pady=(4, 14))

    button_frame = Frame(card, bg="white")
    button_frame.pack(fill="x", pady=(4, 0))

    def run_action(name: str, action) -> None:
        try:
            action()
        except Exception as exc:
            messagebox.showerror("执行失败", str(exc))

    def update_status_label(approved: bool | None) -> None:
        if approved is True:
            status_var.set("当前设备审批状态：已通过")
            status_label.configure(fg="#166534")
        elif approved is False:
            status_var.set("当前设备审批状态：未通过/待审批")
            status_label.configure(fg="#991b1b")
        else:
            status_var.set("当前设备审批状态：同步失败")
            status_label.configure(fg="#92400e")

    def gui_request() -> None:
        request_approval_data(API_BASE, note_var.get())
        messagebox.showinfo("已提交", "已提交待管理员审批。\n同一设备 24 小时内只能提交一次审批。")

    def gui_status() -> None:
        data = refresh_approval_from_api(timeout=10)
        approved = bool(data.get("approved"))
        update_status_label(approved)
        messagebox.showinfo("审批状态", "当前设备审批状态：已通过" if approved else "当前设备审批状态：未通过/待审批")

    def gui_run() -> None:
        args = argparse.Namespace(api=API_BASE)
        run_once(args)
        messagebox.showinfo("执行完成", "受控功能已执行。")

    def gui_install_task() -> None:
        args = argparse.Namespace(api=API_BASE)
        install_task(args)
        messagebox.showinfo("安装完成", f"计划任务已创建/更新：{TASK_NAME}，每 4 分钟执行一次。")

    def gui_remove_task() -> None:
        args = argparse.Namespace(api=None)
        remove_task(args)
        messagebox.showinfo("删除完成", f"计划任务已删除：{TASK_NAME}")

    Button(button_frame, text="提交审批", width=14, command=lambda: run_action("提交审批", gui_request)).grid(row=0, column=0, padx=4, pady=4)
    Button(button_frame, text="同步审批状态", width=14, command=lambda: run_action("同步审批状态", gui_status)).grid(row=0, column=1, padx=4, pady=4)
    Button(button_frame, text="执行一次", width=14, command=lambda: run_action("执行一次", gui_run)).grid(row=0, column=2, padx=4, pady=4)
    Button(button_frame, text="安装计划任务", width=14, command=lambda: run_action("安装计划任务", gui_install_task)).grid(row=1, column=0, padx=4, pady=4)
    Button(button_frame, text="删除计划任务", width=14, command=lambda: run_action("删除计划任务", gui_remove_task)).grid(row=1, column=1, padx=4, pady=4)

    def initial_check() -> None:
        try:
            check_api_reachable(timeout=8)
            status_var.set("当前设备审批状态：同步中")
            try:
                data = refresh_approval_from_api(timeout=10)
                update_status_label(bool(data.get("approved")))
            except Exception:
                update_status_label(None)
        except Exception:
            status_var.set("当前设备审批状态：网络异常")
            status_label.configure(fg="#92400e")
            messagebox.showwarning("网络环境存在问题", "当前网络无法访问审批服务，请更换网络或检查代理后重新打开客户端。")

    root.after(300, initial_check)
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
