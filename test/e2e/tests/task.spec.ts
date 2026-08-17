import { test, expect } from "./fixtures";

test.describe("task running", () => {
  let taskLogPage;

  test.beforeEach(async ({ page, login, project }) => {
    await login(true);

    await project.create("task_runner", true);

    await page.getByTestId("sidebar-templates").click();

    await page.getByRole("link", { name: "Build demo app" }).click();

    await page.getByTestId("template-run").click();

    await page
      .getByTestId("newTaskDialog")
      .getByRole("textbox", { name: "Message (Optional)" })
      .fill("Test");

    const popupPromise = page.waitForEvent("popup");
    await page
      .getByTestId("newTaskDialog")
      .getByTestId("editDialog-save")
      .click();
    taskLogPage = await popupPromise;
    await taskLogPage.waitForLoadState("domcontentloaded");

    test.setTimeout(90000);
  });

  test.afterEach(async ({ page, project }) => {
    if (taskLogPage && !taskLogPage.isClosed()) {
      const closeBtn = taskLogPage
        .getByTestId("taskLogDialog")
        .getByTestId("editDialog-close");
      if (await closeBtn.count()) {
        await closeBtn.click();
      }
      if (!taskLogPage.isClosed()) {
        await taskLogPage.close();
      }
    }

    await project.delete();
  });

  test("run task from demo project", async ({ page }) => {
    await taskLogPage.getByTestId("task-rawLog").waitFor({ timeout: 90000 });

    await expect(taskLogPage.getByTestId("task-status")).toHaveText("Success");
    await expect(page).toHaveURL(/\/templates\//);
  });

  test("stop task on waiting", async () => {
    await taskLogPage
      .getByTestId("taskLogDialog")
      .getByRole("button", { name: "Stop" })
      .click();

    await taskLogPage.getByTestId("task-rawLog").waitFor({ timeout: 600000 });

    await expect(taskLogPage.getByTestId("task-status")).toHaveText("Stopped");
  });

  test("stop task on cloning", async () => {
    await taskLogPage
      .getByTestId("taskLogDialog")
      .getByText("Get current commit hash")
      .waitFor();

    await taskLogPage
      .getByTestId("taskLogDialog")
      .getByRole("button", { name: "Stop" })
      .click();

    await taskLogPage.getByTestId("task-rawLog").waitFor({ timeout: 60000 });

    await expect(taskLogPage.getByTestId("task-status")).toHaveText("Stopped");
  });

  test("stop task on running", async () => {
    await taskLogPage
      .getByTestId("taskLogDialog")
      .getByText(
        "TASK [Gathering Facts] *********************************************************"
      )
      .waitFor({ timeout: 100000 });

    await taskLogPage
      .getByTestId("taskLogDialog")
      .getByRole("button", { name: "Stop" })
      .click();

    await taskLogPage.getByTestId("task-rawLog").waitFor({ timeout: 60000 });

    await expect(taskLogPage.getByTestId("task-status")).toHaveText("Stopped");
  });
});
