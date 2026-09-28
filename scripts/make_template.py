#!/usr/bin/env python3
"""Generates web/assets/employees_template.xlsx, the "Download template"
file offered on Wizard Step 2: a styled Employees sheet with one example
row and a date-formatted join_date column, a Clients sheet listing client
codes, and a dropdown on the client column that reads from it.
"""
import argparse
import datetime
import os

from openpyxl import Workbook
from openpyxl.styles import Alignment, Font, PatternFill
from openpyxl.utils import get_column_letter
from openpyxl.worksheet.datavalidation import DataValidation

HEADER_FILL = PatternFill(start_color="17202A", end_color="17202A", fill_type="solid")
HEADER_FONT = Font(color="FFFFFF", bold=True)

# (column name, width, Excel column index) — order matches the spec's
# "employee_id, name, role, client, join_date, photo_note".
COLUMNS = [
    ("employee_id", 16),
    ("name", 24),
    ("role", 20),
    ("client", 16),
    ("join_date", 14),
    ("photo_note", 26),
]
CLIENT_COL = 4
JOIN_DATE_COL = 5

EXAMPLE_ROW = ["E001", "Priya Rao", "Forklift Operator", "Helios", None, "Wears glasses"]
EXAMPLE_JOIN_DATE = datetime.date(2026, 1, 15)

# Matches the Helios client seeded on first run (internal/clients).
CLIENT_CODES = ["Helios"]

DEFAULT_OUT = os.path.join(
    os.path.dirname(os.path.abspath(__file__)), "..", "web", "assets", "employees_template.xlsx"
)


def style_header(cell):
    cell.font = HEADER_FONT
    cell.fill = HEADER_FILL
    cell.alignment = Alignment(horizontal="left", vertical="center")


def build_workbook():
    wb = Workbook()

    employees = wb.active
    employees.title = "Employees"
    employees.freeze_panes = "A2"

    for col_idx, (name, width) in enumerate(COLUMNS, start=1):
        style_header(employees.cell(row=1, column=col_idx, value=name))
        employees.column_dimensions[get_column_letter(col_idx)].width = width

    for col_idx, value in enumerate(EXAMPLE_ROW, start=1):
        employees.cell(row=2, column=col_idx, value=value)
    join_date_cell = employees.cell(row=2, column=JOIN_DATE_COL, value=EXAMPLE_JOIN_DATE)
    join_date_cell.number_format = "yyyy-mm-dd"

    clients = wb.create_sheet("Clients")
    style_header(clients.cell(row=1, column=1, value="client_code"))
    clients.column_dimensions["A"].width = 20
    for row_idx, code in enumerate(CLIENT_CODES, start=2):
        clients.cell(row=row_idx, column=1, value=code)

    client_col_letter = get_column_letter(CLIENT_COL)
    dv = DataValidation(
        type="list",
        formula1="=Clients!$A$2:$A$1000",
        allow_blank=True,
        showErrorMessage=True,
    )
    dv.errorTitle = "Unknown client"
    dv.error = "Choose a client code from the Clients sheet, or fix it later in the app's Needs attention list."
    employees.add_data_validation(dv)
    dv.add(f"{client_col_letter}2:{client_col_letter}1000")

    return wb


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--out", default=DEFAULT_OUT, help="output .xlsx path")
    args = parser.parse_args()

    out_path = os.path.abspath(args.out)
    os.makedirs(os.path.dirname(out_path), exist_ok=True)
    build_workbook().save(out_path)
    print(f"Wrote {out_path}")


if __name__ == "__main__":
    main()
