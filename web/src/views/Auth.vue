<template>
  <div class="auth">
    <v-dialog v-model="loginHelpDialog" max-width="600">
      <v-card>
        <v-card-title>
          {{ $t('howToFixSigninIssues') }}
          <v-spacer></v-spacer>
          <v-btn icon @click="loginHelpDialog = false">
            <v-icon>mdi-close</v-icon>
          </v-btn>
        </v-card-title>
        <v-card-text>
          <p class="text-body-1">
            {{ $t('firstlyYouNeedAccessToTheServerWhereSemaphoreRunni') }}
          </p>
          <p class="text-body-1">
            {{ $t('executeTheFollowingCommandOnTheServerToSeeExisting') }}
          </p>
          <v-alert
            dense
            text
            color="info"
            style="font-family: monospace;"
          >
            {{ $t('semaphoreUserList') }}
          </v-alert>
          <p class="text-body-1">
            {{ $t('youCanChangePasswordOfExistingUser') }}
          </p>
          <v-alert
            dense
            text
            color="info"
            style="font-family: monospace;"
          >
            {{
              $t('semaphoreUserChangebyloginLoginUser123Password', {
                makePasswordExample:
                  makePasswordExample()
              })
            }}
          </v-alert>
          <p class="text-body-1">
            {{ $t('orCreateNewAdminUser') }}
          </p>
          <v-alert
            dense
            text
            color="info"
            style="font-family: monospace;"
          >
            semaphore user add --admin --login user123 --name User123
            --email user123@example.com --password {{ makePasswordExample() }}
          </v-alert>
        </v-card-text>
        <v-card-actions>
          <v-spacer/>
          <v-btn
            color="blue darken-1"
            text
            @click="loginHelpDialog = false"
          >
            {{ $t('close2') }}
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <v-container
      fluid
      fill-height
      align-center
      justify-center
      class="pa-0"
    >
      <v-card class="px-5 py-5" style="margin-bottom: 10%; border-radius: 15px;">
        <v-card-text>
          <v-form
            ref="signInForm"
            lazy-validation
            v-model="signInFormValid"
            style="width: 350px;"
          >

            <v-img
              width="80"
              height="80"
              transition="0"
              src="favicon.png"
              style="margin: auto;"
              class="mb-4"
            />

            <h2 v-if="screen === 'verification'" class="text-center pt-4 pb-6">
              Two-step verification
            </h2>

            <h2 v-else-if="screen === 'recovery'" class="text-center pt-4 pb-6">
              Account recovery
            </h2>

            <h2 v-else class="text-center pt-4 pb-6">
              Log in to your account
            </h2>

            <v-alert
              :value="signInError != null"
              color="error"
              style="margin-bottom: 20px;"
            >{{ signInError }}
            </v-alert>

            <div v-if="screen === 'verification'">
              <div class="text-center mb-4">
                Open the two-step verification app on your mobile device to
                get your verification code.
              </div>

              <v-otp-input
                v-model="verificationCode"
                length="6"
                @finish="verify()"
              ></v-otp-input>

              <v-divider class="my-6" />

              <div class="text-center">
                <a @click="signOut()" class="mr-6">{{ $t('Return to login') }}</a>
                <a
                  v-if="authMethods.totp && authMethods.totp.allow_recovery"
                  @click="screen = 'recovery'"
                >
                  {{ $t('Use recovery code') }}
                </a>
              </div>
            </div>

            <div v-else-if="screen === 'recovery'">
              <div class="text-center mb-2">
                Use your recovery code to regain access to your account.
              </div>

              <v-text-field
                class="mt-6"
                outlined
                v-model="recoveryCode"
                @keyup.enter.native="signIn"
                :label="$t('Recovery code')"
                :rules="[v => !!v || $t('recoveryCode_required')]"
                required
              />

              <div>
                <v-btn
                  style="width: 100%;"
                  color="primary"
                  @click="recovery()"
                >
                  Send
                </v-btn>
              </div>

              <div class="text-center pt-6">
                <a @click="screen = 'verification'">{{ $t('Return to verification') }}</a>
              </div>

            </div>

            <div v-else>
              <v-text-field
                v-model="username"
                v-bind:label='$t("username")'
                :rules="[v => !!v || $t('username_required')]"
                required
                :disabled="signInProcess"
                v-if="loginWithPassword"
              ></v-text-field>

              <v-text-field
                v-model="password"
                :label="$t('password')"
                :rules="[v => !!v || $t('password_required')]"
                type="password"
                required
                :disabled="signInProcess"
                @keyup.enter.native="signIn"
                style="margin-bottom: 20px;"
                v-if="loginWithPassword"
              ></v-text-field>

              <v-btn
                large
                color="primary"
                @click="signIn"
                :disabled="signInProcess"
                block
                v-if="loginWithPassword"
                rounded
              >
                {{ $t('signIn') }}
              </v-btn>

              <v-btn
                large
                v-for="provider in oidcProviders"
                :color="provider.color || 'secondary'"
                dark
                class="mt-3"
                @click="oidcSignIn(provider.id)"
                block
                :key="provider.id"
                rounded
              >
                <v-icon
                  left
                  dark
                  v-if="provider.icon"
                >
                  mdi-{{ provider.icon }}
                </v-icon>

                {{ provider.name }}
              </v-btn>

              <div class="text-center mt-6" v-if="loginWithPassword">
                <a @click="loginHelpDialog = true">{{ $t('dontHaveAccountOrCantSignIn') }}</a>
              </div>

            </div>
      </v-form>
        </v-card-text>
      </v-card>
    </v-container>
  </div>
</template>
<style lang="scss">
.auth {
  height: 100vh;
  background: #80808024;
}
</style>
<script>
import axios from 'axios';
import { getErrorMessage } from '@/lib/error';

export default {
  data() {
    return {
      signInFormValid: false,
      signInError: null,
      signInProcess: false,

      password: null,
      username: null,

      loginHelpDialog: null,

      oidcProviders: [],
      loginWithPassword: null,
      authMethods: {},

      screen: null,

      verificationCode: null,
      verificationMethod: null,
      recoveryCode: null,
    };
  },

  async created() {
    const { status, verificationMethod } = await this.getAuthenticationStatus();

    switch (status) {
      case 'authenticated':
        document.location = document.baseURI + window.location.search;
        break;
      case 'unauthenticated':
        await this.loadLoginData();
        break;
      case 'unverified':
        this.screen = 'verification';
        this.verificationMethod = verificationMethod;
        await this.loadLoginData();
        break;
      default:
        throw new Error(`Unknown authentication status: ${status}`);
    }
  },

  methods: {
    async loadLoginData() {
      await axios({
        method: 'get',
        url: '/api/auth/login',
        responseType: 'json',
      }).then((resp) => {
        this.oidcProviders = resp.data.oidc_providers;
        this.loginWithPassword = resp.data.login_with_password;
        this.authMethods = resp.data.auth_methods || {};
      });
    },

    async recovery() {
      this.signInProcess = true;

      try {
        await axios({
          method: 'post',
          url: '/api/auth/recovery',
          responseType: 'json',
          data: {
            recovery_code: this.recoveryCode,
          },
        });

        const { location } = document;
        document.location = location;
      } catch (e) {
        this.signInError = getErrorMessage(e);
      } finally {
        this.signInProcess = false;
      }
    },

    async signOut() {
      (await axios({
        method: 'post',
        url: '/api/auth/logout',
        responseType: 'json',
      }));

      const { location } = document;
      document.location = location;
    },

    makePasswordExample() {
      let pwd = '';
      const characters = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789';
      const charactersLength = characters.length;
      for (let i = 0; i < 10; i += 1) {
        pwd += characters.charAt(Math.floor(Math.random() * charactersLength));
      }
      return pwd;
    },

    async getAuthenticationStatus() {
      try {
        await axios({
          method: 'get',
          url: '/api/user',
          responseType: 'json',
        });
      } catch (err) {
        if (err.response.status === 401) {
          switch (err.response.data.error) {
            case 'TOTP_REQUIRED':
              return {
                status: 'unverified',
                verificationMethod: 'totp',
              };
            default:
              return { status: 'unauthenticated' };
          }
        }
        throw err;
      }

      return { status: 'authenticated' };
    },

    async verify() {
      this.signInError = null;

      if (!this.$refs.signInForm.validate()) {
        return;
      }

      this.signInProcess = true;
      try {
        await axios({
          method: 'post',
          url: '/api/auth/verify',
          responseType: 'json',
          data: {
            passcode: this.verificationCode,
          },
        });
        document.location = document.baseURI + window.location.search;
      } catch (err) {
        if (err.response.status === 401) {
          this.signInError = this.$t('Incorrect verification code.');
        } else {
          this.signInError = getErrorMessage(err);
        }
      } finally {
        this.signInProcess = false;
      }
    },

    async signIn() {
      this.signInError = null;

      if (!this.$refs.signInForm.validate()) {
        return;
      }

      this.signInProcess = true;
      try {
        await axios({
          method: 'post',
          url: '/api/auth/login',
          responseType: 'json',
          data: {
            auth: this.username,
            password: this.password,
          },
        });
        document.location = document.baseURI + window.location.search;
      } catch (err) {
        if (err.response.status === 401) {
          this.signInError = this.$t('incorrectUsrPwd');
        } else {
          this.signInError = getErrorMessage(err);
        }
      } finally {
        this.signInProcess = false;
      }
    },

    async oidcSignIn(provider) {
      let query = '';

      if (this.$route.query.new_project === 'premium') {
        query = '?redirect=/project/premium';
      }

      document.location = `${document.baseURI}api/auth/oidc/${provider}/login${query}`;
    },
  },
};
</script>
