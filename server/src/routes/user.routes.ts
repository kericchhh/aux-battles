import { Router } from "express";
import { loginUser, registerUser } from "../handlers/users.js";
import { asyncHandler } from "../utils/AsyncHandler.js";
const router = Router()

router.post("/register", asyncHandler(registerUser))
router.post("/login", asyncHandler(loginUser))

export default router

