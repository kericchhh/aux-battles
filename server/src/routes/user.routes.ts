import { Router } from "express";
import { loginUser, registerUser, getCurrentUser, logoutUser } from "../handlers/users.js";
import { asyncHandler } from "../utils/AsyncHandler.js";
const router = Router()

router.post("/register", asyncHandler(registerUser))
router.post("/login", asyncHandler(loginUser))
router.get("/me", asyncHandler(getCurrentUser))
router.post("/logout", asyncHandler(logoutUser))

export default router

