#!/usr/bin/env python3
"""过滤 DNS 记录 CSV：仅保留 Name 列全部由数字组成的行。

用法:
    python3 filter_dns_numeric_name.py <input.csv> [-o <output.csv>]
    python3 filter_dns_numeric_name.py --self-test

默认输出到 /tmp/dns_records_numeric_<原文件名>.csv
"""

import argparse
import csv
import os
import sys
import tempfile

NAME_COLUMN = "Name"


def is_all_digits(value: str) -> bool:
    """Name 字段整体是否只由数字组成；空字符串不算。"""
    return value != "" and value.isdigit() and value.isascii()


def filter_rows(rows, name_index):
    """返回 Name 为纯数字的行。"""
    kept = []
    for row in rows:
        if name_index >= len(row):
            continue
        if is_all_digits(row[name_index].strip()):
            kept.append(row)
    return kept


def process(input_path: str, output_path: str) -> tuple:
    """读取 CSV 过滤后写出，返回 (保留行数, 丢弃行数)。"""
    with open(input_path, "r", encoding="utf-8-sig", newline="") as fin:
        reader = csv.reader(fin)
        try:
            header = next(reader)
        except StopIteration:
            raise ValueError("输入文件为空")

        if NAME_COLUMN in header:
            name_index = header.index(NAME_COLUMN)
        else:
            # 退化处理：表头找不到时尝试第五列（DNS 记录固定位置）
            if len(header) < 5:
                raise ValueError(
                    "表头中找不到 %s 列，且列数不足 5 列: %r" % (NAME_COLUMN, header)
                )
            name_index = 4

        data = list(reader)
        kept = filter_rows(data, name_index)

    with open(output_path, "w", encoding="utf-8", newline="") as fout:
        writer = csv.writer(fout)
        writer.writerow(header)
        writer.writerows(kept)

    return len(kept), len(data) - len(kept)


def default_output(input_path: str) -> str:
    base = os.path.basename(input_path)
    return os.path.join("/tmp", "dns_records_numeric_" + base)


def self_test() -> None:
    header = ["RecordId", "Value", "Status", "UpdatedOn", "Name",
              "Line", "LineId", "Type", "Weight", "MonitorStatus",
              "Remark", "TTL", "MX", "DefaultNS"]
    samples = [
        # (Name, 是否应保留)
        ("12345", True),
        ("0", True),
        ("@", False),
        ("dataminer", False),
        ("dataminer100", False),
        ("100abc", False),
        ("", False),
        ("   ", False),
        ("12 34", False),
        ("-123", False),
        ("00123", True),
        ("１２３", False),  # 全角数字，isascii() 拦截
    ]

    # 1) is_all_digits 单元断言
    for name, expect in samples:
        got = is_all_digits(name.strip())
        assert got == expect, "is_all_digits(%r) = %r, 期望 %r" % (name, got, expect)

    # 2) 端到端文件处理
    with tempfile.TemporaryDirectory() as tmp:
        src = os.path.join(tmp, "in.csv")
        dst = os.path.join(tmp, "out.csv")
        with open(src, "w", encoding="utf-8", newline="") as f:
            w = csv.writer(f)
            w.writerow(header)
            for i, (name, _) in enumerate(samples):
                w.writerow(["id%d" % i, "v", "ENABLE", "2026-11-05 15:28:20", name,
                            "默认", "0", "A", "", "", "", "600", "0", "false"])

        kept_n, dropped_n = process(src, dst)
        expect_kept = sum(1 for _, ok in samples if ok)

        with open(dst, "r", encoding="utf-8", newline="") as f:
            out_rows = list(csv.reader(f))

        assert out_rows[0] == header, "表头不一致"
        body = out_rows[1:]
        assert len(body) == expect_kept, "保留行数 %d, 期望 %d" % (len(body), expect_kept)
        assert kept_n == expect_kept and dropped_n == len(samples) - expect_kept
        for row in body:
            assert is_all_digits(row[4]), "输出包含非纯数字 Name: %r" % row[4]

    # 3) 表头不含 Name 时回退到第 5 列
    with tempfile.TemporaryDirectory() as tmp:
        src = os.path.join(tmp, "in2.csv")
        dst = os.path.join(tmp, "out2.csv")
        with open(src, "w", encoding="utf-8", newline="") as f:
            w = csv.writer(f)
            w.writerow(["a", "b", "c", "d", "e"])
            w.writerow(["1", "2", "3", "4", "999"])
            w.writerow(["1", "2", "3", "4", "abc"])
        kept_n, dropped_n = process(src, dst)
        assert (kept_n, dropped_n) == (1, 1), "回退列逻辑错误: %d/%d" % (kept_n, dropped_n)

    print("self-test OK: %d 个用例 + 端到端 + 回退列" % len(samples))


def main() -> int:
    parser = argparse.ArgumentParser(description="保留 Name 列为纯数字的 DNS 记录")
    parser.add_argument("input", nargs="?", help="输入 CSV 路径")
    parser.add_argument("-o", "--output", help="输出 CSV 路径")
    parser.add_argument("--self-test", action="store_true", help="运行自检")
    args = parser.parse_args()

    if args.self_test:
        self_test()
        return 0

    if not args.input:
        parser.error("必须提供输入 CSV 路径，或使用 --self-test")

    if not os.path.isfile(args.input):
        print("输入文件不存在: %s" % args.input, file=sys.stderr)
        return 1

    output = args.output or default_output(args.input)
    kept_n, dropped_n = process(args.input, output)
    print("输入: %s" % args.input)
    print("输出: %s" % output)
    print("保留: %d 行, 丢弃: %d 行" % (kept_n, dropped_n))
    return 0


if __name__ == "__main__":
    sys.exit(main())
