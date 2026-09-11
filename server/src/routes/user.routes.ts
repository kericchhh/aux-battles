import { Router } from "express";
import { loginUser, registerUser, getCurrentUser, logoutUser, getProfileHandler } from "../handlers/users.js";
import { asyncHandler } from "../utils/AsyncHandler.js";
import { requireAuth } from "../middleware/authMiddleware.js";
const router = Router()

router.post("/register", asyncHandler(registerUser))
router.post("/login", asyncHandler(loginUser))
router.get("/me", asyncHandler(getCurrentUser))
router.get("/:id",requireAuth, asyncHandler(getProfileHandler))
router.post("/logout", asyncHandler(logoutUser))

export default router

