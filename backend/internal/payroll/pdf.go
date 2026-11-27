package payroll

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/handokodenih/umkm_pos/backend/internal/model"
)

// GeneratePayslipPDF renders a clean A4 PDF 1.4 payslip from an immutable PayrollSnapshot.
func GeneratePayslipPDF(snap *model.PayrollSnapshot) []byte {
	var lines []string
	lines = append(lines, "SLIP GAJI KARYAWAN (PAYSLIP)")
	lines = append(lines, "------------------------------------------------------------")
	if snap.Period != nil {
		lines = append(lines, fmt.Sprintf("Periode      : %s", snap.Period.PeriodName))
		lines = append(lines, fmt.Sprintf("Tgl Bayar    : %s", snap.Period.PaymentDate.Format("2006-01-02")))
	}
	lines = append(lines, fmt.Sprintf("Kode Karyawan: %s", snap.EmployeeCode))
	lines = append(lines, fmt.Sprintf("Nama         : %s", snap.EmployeeName))
	lines = append(lines, fmt.Sprintf("Jabatan      : %s", snap.JobTitle))
	if snap.BankName != "" || snap.BankAccountNumber != "" {
		lines = append(lines, fmt.Sprintf("Rekening     : %s - %s", snap.BankName, snap.BankAccountNumber))
	}
	lines = append(lines, fmt.Sprintf("Status       : %s", snap.Status))
	lines = append(lines, "------------------------------------------------------------")
	lines = append(lines, "PENDAPATAN (EARNINGS):")

	hasEarnings := false
	for _, item := range snap.Items {
		if item.ItemType == "EARNING" {
			hasEarnings = true
			note := ""
			if item.Notes != "" {
				note = " (" + item.Notes + ")"
			}
			lines = append(lines, fmt.Sprintf("  + %-32s Rp %12s%s", item.ComponentName, formatRupiah(item.Amount), note))
		}
	}
	if !hasEarnings {
		lines = append(lines, fmt.Sprintf("  + %-32s Rp %12s", "Gaji Pokok", formatRupiah(snap.BasicSalary)))
	}

	lines = append(lines, "")
	lines = append(lines, "POTONGAN (DEDUCTIONS):")
	hasDeductions := false
	for _, item := range snap.Items {
		if item.ItemType == "DEDUCTION" {
			hasDeductions = true
			note := ""
			if item.Notes != "" {
				note = " (" + item.Notes + ")"
			}
			lines = append(lines, fmt.Sprintf("  - %-32s Rp %12s%s", item.ComponentName, formatRupiah(item.Amount), note))
		}
	}
	if !hasDeductions {
		lines = append(lines, "  (Tidak ada potongan)")
	}

	lines = append(lines, "------------------------------------------------------------")
	lines = append(lines, fmt.Sprintf("Total Tunjangan : Rp %12s", formatRupiah(snap.TotalAllowances)))
	lines = append(lines, fmt.Sprintf("Total Lembur    : Rp %12s", formatRupiah(snap.TotalOvertimePay)))
	lines = append(lines, fmt.Sprintf("Total Komisi    : Rp %12s", formatRupiah(snap.TotalCommissions)))
	lines = append(lines, fmt.Sprintf("Total Potongan  : Rp %12s", formatRupiah(snap.TotalDeductions)))
	lines = append(lines, "============================================================")
	lines = append(lines, fmt.Sprintf("GAJI BERSIH (NET SALARY): Rp %s", formatRupiah(snap.NetSalary)))
	lines = append(lines, "============================================================")
	lines = append(lines, fmt.Sprintf("Snapshot UUID: %s", snap.UUID.String()))

	var content strings.Builder
	content.WriteString("BT\n/F1 10 Tf\n50 780 Td\n14 TL\n")
	for i, line := range lines {
		if i > 0 {
			content.WriteString("T*\n")
		}
		content.WriteString("(")
		content.WriteString(escapePDFText(line))
		content.WriteString(") Tj\n")
	}
	content.WriteString("ET\n")

	stream := content.String()

	var pdf bytes.Buffer
	pdf.WriteString("%PDF-1.4\n")

	offsets := make([]int, 0, 5)

	writeObj := func(id int, body string) {
		offsets = append(offsets, pdf.Len())
		pdf.WriteString(fmt.Sprintf("%d 0 obj\n%s\nendobj\n", id, body))
	}

	writeObj(1, "<< /Type /Catalog /Pages 2 0 R >>")
	writeObj(2, "<< /Type /Pages /Kids [3 0 R] /Count 1 >>")
	writeObj(3, "<< /Type /Page /Parent 2 0 R /MediaBox [0 0 595 842] /Contents 4 0 R /Resources << /Font << /F1 5 0 R >> >> >>")
	writeObj(4, fmt.Sprintf("<< /Length %d >>\nstream\n%sendstream", len(stream), stream))
	writeObj(5, "<< /Type /Font /Subtype /Type1 /BaseFont /Courier >>")

	xrefOffset := pdf.Len()
	pdf.WriteString(fmt.Sprintf("xref\n0 %d\n", len(offsets)+1))
	pdf.WriteString("0000000000 65535 f \n")
	for _, off := range offsets {
		pdf.WriteString(fmt.Sprintf("%010d 00000 n \n", off))
	}
	pdf.WriteString(fmt.Sprintf("trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(offsets)+1, xrefOffset))

	return pdf.Bytes()
}

func escapePDFText(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "(", "\\(")
	s = strings.ReplaceAll(s, ")", "\\)")
	return s
}

func formatRupiah(amount int64) string {
	neg := amount < 0
	if neg {
		amount = -amount
	}
	raw := fmt.Sprintf("%d", amount)
	n := len(raw)
	if n <= 3 {
		if neg {
			return "-" + raw
		}
		return raw
	}
	var out []byte
	rem := n % 3
	if rem > 0 {
		out = append(out, raw[:rem]...)
	}
	for i := rem; i < n; i += 3 {
		if len(out) > 0 {
			out = append(out, '.')
		}
		out = append(out, raw[i:i+3]...)
	}
	if neg {
		return "-" + string(out)
	}
	return string(out)
}
