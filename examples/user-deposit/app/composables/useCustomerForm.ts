export function useCustomerForm() {
  const toast = useToast();

  function showSuccess(message: string) {
    toast.add({
      title: "成功",
      description: message,
      color: "success",
    });
  }

  function showError(error: any, fallback: string) {
    toast.add({
      title: "错误",
      description: error.data?.statusMessage || fallback,
      color: "error",
    });
  }

  return { showSuccess, showError };
}
