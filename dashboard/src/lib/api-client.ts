import axios from "axios";

const apiBaseURL =
  process.env.NEXT_PUBLIC_API_BASE_URL?.trim() || "http://localhost:8080";

// This browser client sends the HttpOnly dashboard cookie; server components need an explicit cookie-forwarding design.
export const apiClient = axios.create({
  baseURL: apiBaseURL,
  withCredentials: true,
});
