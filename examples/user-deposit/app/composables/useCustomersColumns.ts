import type { TableColumn } from "@nuxt/ui";
import type { Customer } from "~/types";
import { formatFeeRate, formatBalance } from "~/utils/customers";

export function useCustomersColumns() {
  const columns: TableColumn<Customer>[] = [
    {
      accessorKey: "label",
      header: "用户标签",
      cell: ({ row }) => {
        return row.original.label || "-";
      },
    },
    {
      accessorKey: "address",
      header: "钱包地址",
    },
    {
      accessorKey: "feeRate",
      header: "充值费率",
      cell: ({ row }) => {
        return h("span", { class: "text-sm" }, formatFeeRate(row.original.feeRate));
      },
    },
    {
      accessorKey: "balance",
      header: "钱包余额",
      cell: ({ row }) => {
        return h("span", { class: "font-medium" }, formatBalance(row.original.balance));
      },
    },
    {
      id: "actions",
      header: "",
    },
  ];

  return { columns };
}
